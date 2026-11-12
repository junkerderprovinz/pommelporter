package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Component is something the installer fetches once the program is in place.
// See pommelporter.Component.
type Component struct {
	ID         string                                                            `json:"id"`
	Name       map[string]string                                                 `json:"name"`
	Target     string                                                            `json:"-"`
	URL        string                                                            `json:"-"`
	SHA256     string                                                            `json:"-"`
	Executable bool                                                              `json:"-"`
	Verify     func(path string) error                                           `json:"-"`
	Fetch      func(ctx context.Context, progress func(done, total int64)) error `json:"-"`
	Platforms  []string                                                          `json:"-"`
}

// The states a component moves through. Present means its Target was there
// already, and failed and skipped ones are left for the app to fetch itself.
const (
	FetchRunning = "running"
	FetchDone    = "done"
	FetchPresent = "present"
	FetchFailed  = "failed"
	FetchSkipped = "skipped"
)

// Fetched is how far one component has come, as the window gets it. Total
// is -1 while the size is unknown. A failed one carries its error in English
// for the details, and a reason the window says in its own words.
type Fetched struct {
	Component string `json:"component"`
	State     string `json:"state"`
	Done      int64  `json:"done,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Error     string `json:"error,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// The reasons a fetch fails for, as the window explains them.
const (
	ReasonNetwork  = "network"
	ReasonServer   = "server"
	ReasonChecksum = "checksum"
	ReasonOther    = "other"
)

// ServerError marks Err as the server's refusal, such as a 404. See
// pommelporter.ServerError.
type ServerError struct{ Err error }

func (e *ServerError) Error() string { return e.Err.Error() }
func (e *ServerError) Unwrap() error { return e.Err }

// ChecksumError marks Err as a download whose digest is not the expected one.
// See pommelporter.ChecksumError.
type ChecksumError struct{ Err error }

func (e *ChecksumError) Error() string { return e.Err.Error() }
func (e *ChecksumError) Unwrap() error { return e.Err }

// reason names why err ended a fetch. A broken connection is told apart by
// the error types the standard library returns for it, so a Fetch of the
// app's own gets that reason without marking anything.
func reason(err error) string {
	var timeout interface{ Timeout() bool }
	switch {
	case errors.As(err, new(*ChecksumError)):
		return ReasonChecksum
	case errors.As(err, new(*ServerError)):
		return ReasonServer
	case errors.As(err, new(*net.OpError)), errors.As(err, new(*net.DNSError)),
		errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, context.DeadlineExceeded),
		errors.As(err, &timeout) && timeout.Timeout():
		return ReasonNetwork
	}
	return ReasonOther
}

// progressEvery spaces out the byte counts, since a large download would
// otherwise send the window an event for every few kilobytes.
var progressEvery = 200 * time.Millisecond

// client fetches the components given by URL. It has no timeout of its own:
// a download of a few hundred megabytes may take long on a slow line, and the
// window's Skip ends it through ctx.
var client = &http.Client{}

// FetchComponents fetches each component in turn and tells how each one goes.
// None of them can fail the install: one that fails is told as failed and the
// next one starts. Once ctx ends, the rest are told as skipped.
func FetchComponents(ctx context.Context, components []Component, tell func(Fetched)) {
	for _, c := range components {
		if ctx.Err() != nil {
			tell(Fetched{Component: c.ID, State: FetchSkipped})
			continue
		}
		if c.Target != "" {
			if _, err := os.Stat(c.Target); err == nil {
				tell(Fetched{Component: c.ID, State: FetchPresent})
				continue
			}
		}
		tell(Fetched{Component: c.ID, State: FetchRunning, Total: -1})
		err := fetchOne(ctx, c, paced(c.ID, tell))
		switch {
		case err == nil:
			tell(Fetched{Component: c.ID, State: FetchDone})
		case ctx.Err() != nil:
			tell(Fetched{Component: c.ID, State: FetchSkipped})
		default:
			tell(Fetched{Component: c.ID, State: FetchFailed, Error: err.Error(), Reason: reason(err)})
		}
	}
}

// paced passes a component's byte counts on at most every progressEvery, and
// the last one of a download whose size is known always.
func paced(id string, tell func(Fetched)) func(done, total int64) {
	var mu sync.Mutex
	var last time.Time
	return func(done, total int64) {
		mu.Lock()
		defer mu.Unlock()
		if done != total && time.Since(last) < progressEvery {
			return
		}
		last = time.Now()
		tell(Fetched{Component: id, State: FetchRunning, Done: done, Total: total})
	}
}

func fetchOne(ctx context.Context, c Component, progress func(done, total int64)) error {
	if c.Fetch != nil {
		return c.Fetch(ctx, progress)
	}
	return download(ctx, c, progress)
}

// download puts the file at c.URL in place as c.Target once its digest and
// c.Verify accept it. It is written beside the target and renamed, so a
// failed or skipped download leaves nothing the app could mistake for its
// copy.
func download(ctx context.Context, c Component, progress func(done, total int64)) error {
	dir := filepath.Dir(c.Target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &ServerError{fmt.Errorf("GET %s: %s", c.URL, resp.Status)}
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(c.Target)+".fetch-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	sum := sha256.New()
	count := &counter{total: resp.ContentLength, report: progress}
	_, copyErr := io.Copy(io.MultiWriter(tmp, sum, count), resp.Body)
	closeErr := tmp.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	if c.SHA256 != "" {
		if got := hex.EncodeToString(sum.Sum(nil)); got != strings.ToLower(c.SHA256) {
			return &ChecksumError{fmt.Errorf("%s has SHA-256 %s instead of %s, so it was thrown away", c.URL, got, c.SHA256)}
		}
	}
	mode := fs.FileMode(0o644)
	if c.Executable {
		mode = 0o755
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	if c.Verify != nil {
		if err := c.Verify(tmp.Name()); err != nil {
			return err
		}
	}
	return os.Rename(tmp.Name(), c.Target)
}

type counter struct {
	done   int64
	total  int64
	report func(done, total int64)
}

func (c *counter) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	c.report(c.done, c.total)
	return len(p), nil
}

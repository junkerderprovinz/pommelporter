package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// runTimeout is how long the program may work on a component given by Run
// before it is ended and the component counts as failed.
var runTimeout = 15 * time.Minute

// diagnostics takes what the program prints on stderr while it works on a
// component, each line led by the component's ID.
var diagnostics io.Writer = os.Stderr

// runProgram runs exe with c.Run in its own folder, with the installer's
// rights and without a window, and passes on the progress and status lines
// it prints.
func runProgram(ctx context.Context, exe string, c Component, tell func(Fetched)) error {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, c.Run...)
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdout = &lineWriter{each: follow(c.ID, tell)}
	cmd.Stderr = &lineWriter{each: func(line string) { fmt.Fprintf(diagnostics, "%s: %s\n", c.ID, line) }}
	// A program it started may hold the output open after it has been ended.
	cmd.WaitDelay = 5 * time.Second
	hideWindow(cmd)
	err := cmd.Run()
	command := filepath.Base(exe) + " " + strings.Join(c.Run, " ")
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s was ended after %v", command, runTimeout)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}
	return nil
}

// follow turns the lines the program prints into the component's progress.
// A status always reaches the window, a fraction at most every
// progressEvery, since a program may print one for every file it unpacks.
func follow(id string, tell func(Fetched)) func(line string) {
	now := Fetched{Component: id, State: FetchRunning}
	var last time.Time
	return func(line string) {
		u, ok := parseLine(line)
		if !ok {
			return
		}
		if u.Status != "" {
			now.Status = u.Status
		} else {
			now.Fraction = u.Fraction
			if time.Since(last) < progressEvery {
				return
			}
		}
		last = time.Now()
		tell(now)
	}
}

// parseLine reads one line the program printed: "progress <fraction>" or
// "status <text>". Any other line is not meant for the window.
func parseLine(line string) (Fetched, bool) {
	word, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
	rest = strings.TrimSpace(rest)
	switch word {
	case "progress":
		f, err := strconv.ParseFloat(rest, 64)
		if err != nil || math.IsNaN(f) {
			return Fetched{}, false
		}
		f = min(max(f, 0), 1)
		return Fetched{Fraction: &f}, true
	case "status":
		if rest == "" {
			return Fetched{}, false
		}
		return Fetched{Status: rest}, true
	}
	return Fetched{}, false
}

// lineWriter hands each line written to it to each, without its line ending.
type lineWriter struct {
	each func(string)
	rest []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.rest = append(w.rest, p...)
	for {
		i := bytes.IndexByte(w.rest, '\n')
		if i < 0 {
			return len(p), nil
		}
		w.each(strings.TrimSuffix(string(w.rest[:i]), "\r"))
		w.rest = w.rest[i+1:]
	}
}

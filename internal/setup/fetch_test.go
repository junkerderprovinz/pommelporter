package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// serve answers every request with body over TLS and points the downloads at
// that server for the test.
func serve(t *testing.T, body []byte) string {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	was := client
	client = srv.Client()
	t.Cleanup(func() { client = was })
	return srv.URL
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// states runs FetchComponents and returns each component's states in order.
func states(ctx context.Context, components []Component) (map[string][]string, map[string]string) {
	got, errs, _ := fetchAll(ctx, components)
	return got, errs
}

// fetchAll is states with the reason each failed component ended with.
func fetchAll(ctx context.Context, components []Component) (map[string][]string, map[string]string, map[string]string) {
	got := map[string][]string{}
	errs := map[string]string{}
	reasons := map[string]string{}
	FetchComponents(ctx, components, func(f Fetched) {
		if n := len(got[f.Component]); n > 0 && got[f.Component][n-1] == f.State {
			return
		}
		got[f.Component] = append(got[f.Component], f.State)
		if f.Error != "" {
			errs[f.Component] = f.Error
			reasons[f.Component] = f.Reason
		}
	})
	return got, errs, reasons
}

func TestADownloadThatMatchesItsDigestLandsAtItsTarget(t *testing.T) {
	body := []byte("a codec pack")
	url := serve(t, body)
	target := filepath.Join(t.TempDir(), "tools", "codecs.bin")

	got, _ := states(context.Background(), []Component{{ID: "codecs", Target: target, URL: url + "/codecs.bin", SHA256: digest(body)}})
	if want := []string{FetchRunning, FetchDone}; !reflect.DeepEqual(got["codecs"], want) {
		t.Fatalf("states %v, want %v", got["codecs"], want)
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != string(body) {
		t.Fatalf("target holds %q, %v", b, err)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".*")); len(left) > 0 {
		t.Fatalf("left behind %v", left)
	}
}

func TestADownloadWithTheWrongDigestIsThrownAwayAndTheNextOneStillRuns(t *testing.T) {
	url := serve(t, []byte("tampered"))
	dir := t.TempDir()
	bad := filepath.Join(dir, "runtime.bin")
	good := filepath.Join(dir, "plugin.bin")

	got, errs := states(context.Background(), []Component{
		{ID: "runtime", Target: bad, URL: url + "/runtime.bin", SHA256: digest([]byte("genuine"))},
		{ID: "plugin", Target: good, URL: url + "/plugin.bin", SHA256: digest([]byte("tampered"))},
	})
	if got["runtime"][len(got["runtime"])-1] != FetchFailed || errs["runtime"] == "" {
		t.Fatalf("runtime ended %v with %q", got["runtime"], errs["runtime"])
	}
	if _, err := os.Stat(bad); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refused download is at its target: %v", err)
	}
	if got["plugin"][len(got["plugin"])-1] != FetchDone {
		t.Fatalf("plugin ended %v", got["plugin"])
	}
}

func TestVerifyHasTheLastWordOnADownload(t *testing.T) {
	url := serve(t, []byte("not a program"))
	target := filepath.Join(t.TempDir(), "helper")
	var checked string
	refuse := func(path string) error {
		checked = path
		return errors.New("does not run here")
	}

	got, errs := states(context.Background(), []Component{{ID: "helper", Target: target, URL: url + "/helper", Verify: refuse}})
	if got["helper"][len(got["helper"])-1] != FetchFailed || errs["helper"] != "does not run here" {
		t.Fatalf("helper ended %v with %q", got["helper"], errs["helper"])
	}
	if checked == "" || checked == target {
		t.Fatalf("Verify saw %q, not the download beside the target", checked)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refused download is at its target: %v", err)
	}
}

func TestAMissingFileOnTheServerFailsTheComponent(t *testing.T) {
	url := serve(t, nil)
	got, errs := states(context.Background(), []Component{{ID: "x", Target: filepath.Join(t.TempDir(), "x"), URL: url + "/missing", SHA256: digest(nil)}})
	if got["x"][len(got["x"])-1] != FetchFailed || errs["x"] == "" {
		t.Fatalf("x ended %v with %q", got["x"], errs["x"])
	}
}

func TestAComponentWhoseTargetExistsIsLeftAsItIs(t *testing.T) {
	target := filepath.Join(t.TempDir(), "runtime")
	if err := os.WriteFile(target, []byte("the program's own newer copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	ran := false
	got, _ := states(context.Background(), []Component{{ID: "runtime", Target: target, Fetch: func(context.Context, func(int64, int64)) error {
		ran = true
		return nil
	}}})
	if ran || !reflect.DeepEqual(got["runtime"], []string{FetchPresent}) {
		t.Fatalf("ran %v, states %v", ran, got["runtime"])
	}
}

func TestFetchReportsItsProgressAndItsEnd(t *testing.T) {
	was := progressEvery
	progressEvery = 0
	t.Cleanup(func() { progressEvery = was })

	var seen []Fetched
	FetchComponents(context.Background(), []Component{{ID: "pack", Fetch: func(_ context.Context, progress func(int64, int64)) error {
		progress(50, 100)
		progress(100, 100)
		return nil
	}}}, func(f Fetched) { seen = append(seen, f) })

	want := []Fetched{
		{Component: "pack", State: FetchRunning, Total: -1},
		{Component: "pack", State: FetchRunning, Done: 50, Total: 100},
		{Component: "pack", State: FetchRunning, Done: 100, Total: 100},
		{Component: "pack", State: FetchDone},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("got %+v", seen)
	}
}

func TestSkippingStopsTheRunningFetchAndLeavesTheRestForLater(t *testing.T) {
	ctx, skip := context.WithCancel(context.Background())
	defer skip()
	waits := func(ctx context.Context, _ func(int64, int64)) error {
		skip()
		<-ctx.Done()
		return ctx.Err()
	}
	never := func(context.Context, func(int64, int64)) error {
		t.Error("a component after the skip was fetched")
		return nil
	}

	got, _ := states(ctx, []Component{{ID: "big", Fetch: waits}, {ID: "small", Fetch: never}})
	if want := []string{FetchRunning, FetchSkipped}; !reflect.DeepEqual(got["big"], want) {
		t.Fatalf("big %v, want %v", got["big"], want)
	}
	if want := []string{FetchSkipped}; !reflect.DeepEqual(got["small"], want) {
		t.Fatalf("small %v, want %v", got["small"], want)
	}
}

func TestAFailureIsExplainedByItsCause(t *testing.T) {
	url := serve(t, []byte("tampered"))
	dir := t.TempDir()
	fails := func(err error) func(context.Context, func(int64, int64)) error {
		return func(context.Context, func(int64, int64)) error { return err }
	}
	offline := fmt.Errorf("downloading the runtime: %w", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")})

	_, _, got := fetchAll(context.Background(), []Component{
		{ID: "missing", Target: filepath.Join(dir, "a"), URL: url + "/missing", SHA256: digest(nil)},
		{ID: "tampered", Target: filepath.Join(dir, "b"), URL: url + "/b", SHA256: digest([]byte("genuine"))},
		{ID: "offline", Fetch: fails(offline)},
		{ID: "marked", Fetch: fails(&ServerError{errors.New("github answered 503")})},
		{ID: "broken", Fetch: fails(errors.New("the archive holds no bin/ffmpeg"))},
	})
	want := map[string]string{
		"missing":  ReasonServer,
		"tampered": ReasonChecksum,
		"offline":  ReasonNetwork,
		"marked":   ReasonServer,
		"broken":   ReasonOther,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reasons %v, want %v", got, want)
	}
}

func TestComponentsRunOnlyWhereTheyApply(t *testing.T) {
	app := App{Components: []Component{{ID: "both"}, {ID: "linux", Platforms: []string{"linux"}}}}
	var ids []string
	for _, c := range app.ComponentsOn("windows") {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []string{"both"}) {
		t.Fatalf("on Windows %v", ids)
	}
}

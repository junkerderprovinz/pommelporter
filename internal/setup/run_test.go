package setup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// playProgram makes the test binary stand in for the installed program; its
// first argument picks what the program does.
const playProgram = "POMMELPORTER_TEST_PROGRAM"

func TestMain(m *testing.M) {
	if os.Getenv(playProgram) != "" {
		os.Exit(play(os.Args[1]))
	}
	os.Exit(m.Run())
}

func play(what string) int {
	switch what {
	case "report":
		wd, _ := os.Getwd()
		fmt.Print("progress 0.25\r\n")
		fmt.Println("unpacking part 2")
		fmt.Println("status " + wd)
		fmt.Fprintln(os.Stderr, "a warning")
		fmt.Println("progress 1")
	case "chatty":
		fmt.Println("progress 0.1")
		fmt.Println("progress 0.2")
		fmt.Println("status Part 2 of 3")
		fmt.Println("progress 0.3")
	case "fail":
		fmt.Println("progress 0.5")
		return 3
	case "hang":
		fmt.Println("status Waiting")
		time.Sleep(time.Minute)
	}
	return 0
}

// program is the test binary as the installed program, for the test.
func program(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(playProgram, "1")
	return self
}

// everyLine lets every progress line through for the test.
func everyLine(t *testing.T) {
	was := progressEvery
	progressEvery = 0
	t.Cleanup(func() { progressEvery = was })
}

func fraction(f float64) *float64 { return &f }

func TestReadingWhatTheProgramPrints(t *testing.T) {
	for _, c := range []struct {
		line string
		want Fetched
		ok   bool
	}{
		{"progress 0.4", Fetched{Fraction: fraction(0.4)}, true},
		{"progress 0", Fetched{Fraction: fraction(0)}, true},
		{"  progress 1  ", Fetched{Fraction: fraction(1)}, true},
		{"progress 1.5", Fetched{Fraction: fraction(1)}, true},
		{"progress -2", Fetched{Fraction: fraction(0)}, true},
		{"status Fetching part 2 of 3", Fetched{Status: "Fetching part 2 of 3"}, true},
		{"progress", Fetched{}, false},
		{"progress half", Fetched{}, false},
		{"progress NaN", Fetched{}, false},
		{"status", Fetched{}, false},
		{"status   ", Fetched{}, false},
		{"Progress 0.4", Fetched{}, false},
		{"progressing 0.4", Fetched{}, false},
		{"fetched 3 of 7 parts", Fetched{}, false},
		{"", Fetched{}, false},
	} {
		got, ok := parseLine(c.line)
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %+v, %t", c.line, got, ok)
		}
	}
}

func TestLinesAreHandedOnWholeWhateverTheChunks(t *testing.T) {
	var lines []string
	w := &lineWriter{each: func(s string) { lines = append(lines, s) }}
	for _, chunk := range []string{"prog", "ress 0.5\r\nstatus a", "\n\nstatus b\n", "unfinished"} {
		w.Write([]byte(chunk))
	}
	if want := []string{"progress 0.5", "status a", "", "status b"}; !reflect.DeepEqual(lines, want) {
		t.Fatalf("lines %q", lines)
	}
}

func TestTheProgramRunsInItsFolderAndItsLinesReachTheRow(t *testing.T) {
	everyLine(t)
	exe := program(t)
	var log bytes.Buffer
	was := diagnostics
	diagnostics = &log
	t.Cleanup(func() { diagnostics = was })

	var seen []Fetched
	FetchComponents(context.Background(), exe, []Component{{ID: "extras", Run: []string{"report"}}}, func(f Fetched) { seen = append(seen, f) })

	if len(seen) != 5 {
		t.Fatalf("got %+v", seen)
	}
	if seen[1].Fraction == nil || *seen[1].Fraction != 0.25 || seen[1].Status != "" {
		t.Errorf("second %+v, want progress 0.25", seen[1])
	}
	wd, err := os.Stat(seen[2].Status)
	if err != nil {
		t.Fatalf("status %q: %v", seen[2].Status, err)
	}
	if dir, _ := os.Stat(filepath.Dir(exe)); !os.SameFile(wd, dir) {
		t.Errorf("the program ran in %q, want %q", seen[2].Status, filepath.Dir(exe))
	}
	if *seen[2].Fraction != 0.25 {
		t.Errorf("third %+v lost the fraction", seen[2])
	}
	if last := seen[3]; last.Fraction == nil || *last.Fraction != 1 || last.Status != seen[2].Status {
		t.Errorf("fourth %+v, want progress 1 under the same status", last)
	}
	if seen[4].State != FetchDone {
		t.Errorf("last %+v, want done", seen[4])
	}
	if !strings.Contains(log.String(), "extras: a warning\n") {
		t.Errorf("diagnostics %q lack the program's stderr", log.String())
	}
}

func TestAStatusAlwaysGetsThroughWhileFractionsAreSpacedOut(t *testing.T) {
	was := progressEvery
	progressEvery = time.Hour
	t.Cleanup(func() { progressEvery = was })

	var seen []Fetched
	FetchComponents(context.Background(), program(t), []Component{{ID: "extras", Run: []string{"chatty"}}}, func(f Fetched) { seen = append(seen, f) })

	want := []Fetched{
		{Component: "extras", State: FetchRunning, Total: -1},
		{Component: "extras", State: FetchRunning, Fraction: fraction(0.1)},
		{Component: "extras", State: FetchRunning, Fraction: fraction(0.2), Status: "Part 2 of 3"},
		{Component: "extras", State: FetchDone},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("got %+v", seen)
	}
}

func TestAProgramThatFailsLeavesItsComponentForLater(t *testing.T) {
	everyLine(t)
	exe := program(t)
	got, errs, reasons := fetchAll(context.Background(), exe, []Component{
		{ID: "first", Run: []string{"fail"}},
		{ID: "second", Run: []string{"quiet"}},
	})
	if want := []string{FetchRunning, FetchFailed}; !reflect.DeepEqual(got["first"], want) {
		t.Fatalf("first %v, want %v", got["first"], want)
	}
	if !strings.Contains(errs["first"], "fail: exit status 3") || reasons["first"] != ReasonOther {
		t.Errorf("first failed with %q, %q", errs["first"], reasons["first"])
	}
	if want := []string{FetchRunning, FetchDone}; !reflect.DeepEqual(got["second"], want) {
		t.Errorf("second %v, want %v", got["second"], want)
	}
}

func TestAProgramThatIsMissingLeavesItsComponentForLater(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "missing")
	got, errs, reasons := fetchAll(context.Background(), exe, []Component{{ID: "extras", Run: []string{"--extras"}}})
	if want := []string{FetchRunning, FetchFailed}; !reflect.DeepEqual(got["extras"], want) || errs["extras"] == "" || reasons["extras"] != ReasonOther {
		t.Fatalf("extras %v with %q, %q", got["extras"], errs["extras"], reasons["extras"])
	}
}

func TestAProgramThatRunsTooLongIsEnded(t *testing.T) {
	was := runTimeout
	runTimeout = 300 * time.Millisecond
	t.Cleanup(func() { runTimeout = was })

	began := time.Now()
	got, errs, reasons := fetchAll(context.Background(), program(t), []Component{{ID: "slow", Run: []string{"hang"}}})
	if want := []string{FetchRunning, FetchFailed}; !reflect.DeepEqual(got["slow"], want) {
		t.Fatalf("slow %v, want %v", got["slow"], want)
	}
	if !strings.Contains(errs["slow"], "was ended after 300ms") || reasons["slow"] != ReasonOther {
		t.Errorf("slow failed with %q, %q", errs["slow"], reasons["slow"])
	}
	if took := time.Since(began); took > 30*time.Second {
		t.Errorf("ending the program took %v", took)
	}
}

func TestSkippingEndsTheProgram(t *testing.T) {
	ctx, skip := context.WithCancel(context.Background())
	defer skip()
	exe := program(t)

	began := time.Now()
	var states []string
	FetchComponents(ctx, exe, []Component{{ID: "slow", Run: []string{"hang"}}}, func(f Fetched) {
		if f.Status != "" {
			skip()
		}
		if n := len(states); n == 0 || states[n-1] != f.State {
			states = append(states, f.State)
		}
	})
	if want := []string{FetchRunning, FetchSkipped}; !reflect.DeepEqual(states, want) {
		t.Fatalf("states %v, want %v", states, want)
	}
	if took := time.Since(began); took > 30*time.Second {
		t.Errorf("ending the program took %v", took)
	}
}

func TestAProgramWhoseTargetExistsIsNotRun(t *testing.T) {
	target := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, _ := fetchAll(context.Background(), filepath.Join(t.TempDir(), "missing"), []Component{{ID: "helper", Target: target, Run: []string{"--helper"}}})
	if !reflect.DeepEqual(got["helper"], []string{FetchPresent}) {
		t.Fatalf("helper %v", got["helper"])
	}
}

//go:build windows

package windows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// playUninstaller makes the test program stand in for uninstall.exe: it hands
// over as the real one does, and the copy finishes with code 3.
const playUninstaller = "POMMELPORTER_TEST_UNINSTALLER"

func TestMain(m *testing.M) {
	if os.Getenv(playUninstaller) != "" {
		if handled, code, err := HandOver(os.Args[1:]); err != nil {
			os.Exit(2)
		} else if handled {
			os.Exit(code)
		}
		FinishHandOver(3)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestTheUninstallerLeavesNeitherItselfNorItsFolder(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "Prog")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	un := filepath.Join(dir, uninstaller)
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(un, b, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(un, "--uninstall", "/S")
	temp := t.TempDir()
	cmd.Env = append(os.Environ(), playUninstaller+"=1", "TMP="+temp, "TEMP="+temp)
	err = cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 3 {
		t.Fatalf("the uninstaller ended with %v, want the copy's code 3", err)
	}
	// The copy removes them a moment after the original ends.
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			entries, _ := os.ReadDir(dir)
			t.Fatalf("the folder is still there with %v", entries)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestTheCopyReportsItsCode(t *testing.T) {
	for line, want := range map[string]int{"done 0\n": 0, "done 1\n": 1, "done 3": 3} {
		if code, ok := readDone(strings.NewReader(line)); !ok || code != want {
			t.Errorf("%q: got %d %v", line, code, ok)
		}
	}
	for _, line := range []string{"", "hello\n", "done x\n"} {
		if _, ok := readDone(strings.NewReader(line)); ok {
			t.Errorf("%q counted as done", line)
		}
	}
}

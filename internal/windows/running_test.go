//go:build windows

package windows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// playParent makes the test program stand in for an app that has started a
// program of its own and keeps running.
const playParent = "POMMELPORTER_TEST_PARENT"

func startsAChild() {
	child := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "ping.exe"), "-n", "60", "127.0.0.1")
	if err := child.Start(); err != nil {
		os.Exit(2)
	}
	time.Sleep(60 * time.Second)
	os.Exit(0)
}

// childrenOf lists the processes whose parent is pid.
func childrenOf(t *testing.T, pid uint32) []uint32 {
	t.Helper()
	var kids []uint32
	for _, p := range processes() {
		if p.parent == pid {
			kids = append(kids, p.pid)
		}
	}
	return kids
}

func alive(pid uint32) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	ev, _ := windows.WaitForSingleObject(h, 0)
	return ev == uint32(windows.WAIT_TIMEOUT)
}

func TestEndRunningEndsTheProgramAndWhatItStarted(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "Prog.exe")
	if err := os.WriteFile(exe, b, 0o755); err != nil {
		t.Fatal(err)
	}
	app := exec.Command(exe)
	app.Env = append(os.Environ(), playParent+"=1")
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Process.Kill(); app.Wait() })
	elsewhere := filepath.Join(t.TempDir(), "Prog.exe")
	running(t, elsewhere)

	var kids []uint32
	for deadline := time.Now().Add(10 * time.Second); len(kids) == 0 && time.Now().Before(deadline); {
		time.Sleep(100 * time.Millisecond)
		kids = childrenOf(t, uint32(app.Process.Pid))
	}
	if len(kids) == 0 {
		t.Fatal("the stand-in app started no program")
	}

	endRunning(exe)

	if alive(uint32(app.Process.Pid)) {
		t.Error("the app still runs")
	}
	for _, k := range kids {
		if alive(k) {
			t.Errorf("process %d the app started still runs", k)
		}
	}
	var others int
	for _, p := range processes() {
		if strings.EqualFold(p.image, elsewhere) {
			others++
		}
	}
	if others != 1 {
		t.Errorf("%d copies run from %s, want the one somebody else started", others, elsewhere)
	}
}

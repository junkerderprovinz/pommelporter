//go:build windows

package windows

import (
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// runAsPerson starts exe with args through Explorer, so it changes the
// settings of the person signed in rather than the installer's, and waits
// for it to end. Explorer hands back no process, so the wait watches for a
// process of that program to come and go.
func runAsPerson(exe string, args []string) error {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = syscall.EscapeArg(a)
	}
	if err := shellExecute(exe, strings.Join(quoted, " ")); err != nil {
		return err
	}
	waitForProgram(exe, 5*time.Second, 30*time.Second)
	return nil
}

// waitForProgram waits up to start for a process of exe to appear, then up
// to end for every one of them to be gone.
func waitForProgram(exe string, start, end time.Duration) {
	seen := false
	began := time.Now()
	for {
		running := countProcesses(exe) > 0
		seen = seen || running
		switch {
		case seen && !running:
			return
		case !seen && time.Since(began) > start:
			return
		case time.Since(began) > start+end:
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// countProcesses counts the running processes whose image is exe.
func countProcesses(exe string) int {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	n := 0
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, e.ProcessID)
		if err != nil {
			continue
		}
		if strings.EqualFold(imageName(proc), exe) {
			n++
		}
		windows.CloseHandle(proc)
	}
	return n
}

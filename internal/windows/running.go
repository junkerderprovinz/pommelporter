//go:build windows

package windows

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	procFindWindowExW = user32.NewProc("FindWindowExW")
	procPostMessageW  = user32.NewProc("PostMessageW")
)

const (
	hwndMessage = ^uintptr(2) // HWND_MESSAGE, -3
	wmClose     = 0x10
	// closeWait covers the 15 seconds the apps give a run under way to stop.
	closeWait = 30_000
)

func findWindow(parent, after uintptr, class, title string) uintptr {
	c, _ := windows.UTF16PtrFromString(class)
	t, _ := windows.UTF16PtrFromString(title)
	hwnd, _, _ := procFindWindowExW.Call(parent, after, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)))
	return hwnd
}

// closeRunning asks a copy running from dir to quit as its own Quit would,
// so a run under way stops and records itself, and waits for it to end.
// Every Wails program has a hidden main-thread window that quits it when
// closed; the single-instance lock's window tells which process is the app.
// It reports whether a copy quit.
func closeRunning(dir, instanceID string) bool {
	if instanceID == "" {
		return false
	}
	id := "wails-app-" + instanceID
	lock := findWindow(hwndMessage, 0, id+"-sic", id+"-siw")
	if lock == 0 {
		return false
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(windows.HWND(lock), &pid); err != nil {
		return false
	}
	proc, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(proc)
	// A portable copy elsewhere is somebody else's to close.
	if !strings.EqualFold(filepath.Dir(imageName(proc)), dir) {
		return false
	}

	for w := findWindow(0, 0, "WailsWebviewWindow", "__wails_hidden_mainthread"); w != 0; w = findWindow(0, w, "WailsWebviewWindow", "__wails_hidden_mainthread") {
		var owner uint32
		if _, err := windows.GetWindowThreadProcessId(windows.HWND(w), &owner); err != nil || owner != pid {
			continue
		}
		procPostMessageW.Call(w, wmClose, 0, 0)
		ev, _ := windows.WaitForSingleObject(proc, closeWait)
		return ev == windows.WAIT_OBJECT_0
	}
	return false
}

func imageName(proc windows.Handle) string {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(proc, 0, &buf[0], &n); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

// stepAside gets exe out of the way of the new one. A copy that still runs
// cannot be overwritten but can be renamed: it becomes .old, as it does for
// an update, and the next update removes it. When .old is held too, it takes
// a name the updater's cleanup removes as well.
func stepAside(exe string) error {
	old := exe + ".old"
	err := os.Remove(exe)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		os.Remove(old)
		return nil
	}
	if err := os.Remove(old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		aside, err := os.CreateTemp(filepath.Dir(exe), "."+filepath.Base(exe)+".update-*")
		if err != nil {
			return err
		}
		aside.Close()
		old = aside.Name()
	}
	return os.Rename(exe, old)
}

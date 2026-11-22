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
	// closeWait leaves room for an app that takes up to 15 seconds to stop
	// a run under way.
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

// process is one entry of a process snapshot.
type process struct {
	pid, parent uint32
	image       string
}

// processes lists the running processes with the image each was started
// from, "" where this process may not ask.
func processes() []process {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)
	var out []process
	e := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err := windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		p := process{pid: e.ProcessID, parent: e.ParentProcessID}
		if h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, e.ProcessID); err == nil {
			p.image = imageName(h)
			windows.CloseHandle(h)
		}
		out = append(out, p)
	}
	return out
}

// endRunning ends a copy of exe that closeRunning could not reach, together
// with the programs it started. Its window lies on a desktop this process
// cannot see when the copy runs in another session, as it does when a
// deployment tool uninstalls as the system account or someone does over SSH,
// and a copy left running keeps its files and those of its helpers open.
func endRunning(exe string) {
	all := processes()
	for _, p := range all {
		if p.image != "" && strings.EqualFold(p.image, exe) {
			endTree(p.pid, all)
		}
	}
}

// endTree ends pid after the processes it started. A process whose parent
// id is pid but which is older than pid had another parent of that id.
func endTree(pid uint32, all []process) {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	born := started(h)
	for _, p := range all {
		if p.parent != pid || p.pid == pid {
			continue
		}
		if c, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, p.pid); err == nil {
			older := started(c) < born
			windows.CloseHandle(c)
			if !older {
				endTree(p.pid, all)
			}
		}
	}
	windows.TerminateProcess(h, 1)
	windows.WaitForSingleObject(h, 5000)
}

// started is when the process began, in 100-nanosecond steps.
func started(h windows.Handle) int64 {
	var created, exited, kernel, user windows.Filetime
	if windows.GetProcessTimes(h, &created, &exited, &kernel, &user) != nil {
		return 0
	}
	return created.Nanoseconds() / 100
}

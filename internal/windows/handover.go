//go:build windows

package windows

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// The uninstaller in the install folder cannot delete its own file while it
// runs. As NSIS does, it hands the work to a copy of itself in the temp
// folder and waits for it, so whoever waits for the uninstall still waits
// for all of it. The copy removes the folder and, once the original has
// ended, the original too; Windows deletes the copy at the next start.
const (
	originalEnv = "POMMELPORTER_UNINSTALLER"
	parentEnv   = "POMMELPORTER_PARENT"
)

// handedOver is the path of the uninstaller this copy works for, or "".
func handedOver() string { return os.Getenv(originalEnv) }

// HandOver runs an uninstall from a copy in the temp folder when the running
// program is the uninstaller in its install folder. When it did, the caller
// exits with code.
func HandOver(args []string) (handled bool, code int, err error) {
	if handedOver() != "" {
		return false, 0, nil
	}
	self, err := os.Executable()
	if err != nil || !strings.EqualFold(filepath.Base(self), uninstaller) {
		return false, 0, nil
	}
	copied, err := copyToTemp(self)
	if err != nil {
		// Without a copy the uninstall still runs; only its file stays
		// until the next start.
		return false, 0, nil
	}
	_ = removeAtRestart(copied)
	// A working folder inside the install folder would keep it from going.
	_ = os.Chdir(os.TempDir())

	cmd := exec.Command(copied, args...)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), originalEnv+"="+self, parentEnv+"="+strconv.Itoa(os.Getpid()))
	r, w, err := os.Pipe()
	if err != nil {
		return false, 0, nil
	}
	cmd.Stdout = w
	if err := cmd.Start(); err != nil {
		w.Close()
		r.Close()
		return false, 0, nil
	}
	w.Close()
	code, ok := readDone(r)
	r.Close()
	if !ok {
		// The copy ended without a word, so its exit code tells.
		err := cmd.Wait()
		if exit, isExit := err.(*exec.ExitError); isExit {
			return true, exit.ExitCode(), nil
		}
		return true, 0, err
	}
	go cmd.Wait()
	return true, code, nil
}

// readDone reads the copy's "done <code>" line.
func readDone(r io.Reader) (int, bool) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && line == "" {
		return 0, false
	}
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), "done ")
	if !ok {
		return 0, false
	}
	code, err := strconv.Atoi(rest)
	return code, err == nil
}

// FinishHandOver tells the waiting original that the work is done, waits for
// it to end and removes it and its folder, which is empty then.
func FinishHandOver(code int) {
	original := handedOver()
	if original == "" {
		return
	}
	fmt.Fprintf(os.Stdout, "done %d\n", code)
	os.Stdout.Close()
	if pid, err := strconv.Atoi(os.Getenv(parentEnv)); err == nil {
		waitForExit(uint32(pid), 30*time.Second)
	}
	// Windows lets go of a program's file a moment after the program ends.
	for range 50 {
		if err := os.Remove(original); err == nil || errors.Is(err, fs.ErrNotExist) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for range 50 {
		if err := os.Remove(filepath.Dir(original)); err == nil || errors.Is(err, fs.ErrNotExist) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func waitForExit(pid uint32, within time.Duration) {
	proc, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(proc)
	windows.WaitForSingleObject(proc, uint32(within.Milliseconds()))
}

func copyToTemp(self string) (string, error) {
	src, err := os.Open(self)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.CreateTemp("", strings.TrimSuffix(uninstaller, ".exe")+"-*.exe")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dst.Name())
		return "", err
	}
	return dst.Name(), dst.Close()
}

//go:build windows

package windows

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// removeTree deletes path and what is below it. A link anywhere in it is
// removed as a link: the installer runs as an administrator, and following a
// junction a user planted in their own folder would delete wherever it leads.
func removeTree(path string) error { return removeTreeWith(path, os.Remove, os.Remove) }

// removeTreeSoon is removeTree for files a running program may still hold,
// which go at the next start together with the folders they are in.
func removeTreeSoon(path string) error { return removeTreeWith(path, removeSoon, removeDirSoon) }

func removeTreeWith(path string, file, dir func(string) error) error {
	attrs, err := attributes(path)
	if notFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if isLink(attrs) || !isDir(attrs) {
		return file(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var first error
	for _, e := range entries {
		if err := removeTreeWith(filepath.Join(path, e.Name()), file, dir); err != nil && first == nil {
			first = err
		}
	}
	if first != nil {
		return first
	}
	return dir(path)
}

// removeSoon deletes the file at path, or has Windows delete it at the next
// start when a running program still holds it.
func removeSoon(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return removeAtRestart(path)
}

func removeAtRestart(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(p, nil, windows.MOVEFILE_DELAY_UNTIL_REBOOT)
}

// removeDirSoon removes dir when it is empty, or else asks Windows to try
// again at the next start, after the files waiting for that are gone. A
// folder that still holds other files then stays.
func removeDirSoon(dir string) error {
	err := os.Remove(dir)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return removeAtRestart(dir)
}

// removeMatches deletes what matches pattern, each one as removeTree does.
func removeMatches(pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	var first error
	for _, m := range matches {
		if err := removeTree(m); err != nil && first == nil {
			first = err
		}
	}
	return first
}

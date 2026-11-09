//go:build linux

package linux

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// standIn writes the hidden entry of the open window, or is nil without one.
// Uninstall calls it before it takes the app's own entry away, so the window
// never stands without one.
var standIn struct {
	sync.Mutex
	write func()
}

// WindowEntry gives the installer's window a hidden menu entry whenever the
// app has none of its own, and returns what removes it again. GNOME takes
// the icon of a window only from the entry its WM_CLASS names and shows a
// gear for any other. While the app is installed, its own entry names the
// class.
func (platform) WindowEntry(app setup.App) (remove func()) {
	remove = func() {}
	p, err := locate(app)
	if err != nil || len(app.Icon) == 0 {
		return remove
	}
	exe, err := os.Executable()
	if err != nil {
		return remove
	}
	// The icon is copied before anything else, since an uninstall takes the
	// installed one away.
	dir, err := os.MkdirTemp("", "pommelporter-")
	if err != nil {
		return remove
	}
	icon := filepath.Join(dir, "icon.png")
	if writeFile(icon, bytes.NewReader(app.Icon), 0o644) != nil {
		os.RemoveAll(dir)
		return remove
	}
	entry := strings.TrimSuffix(p.entry, ".desktop") + "-setup.desktop"
	body := "[Desktop Entry]\nType=Application\nName=" + escapeValue(app.Name) +
		"\nExec=" + execArg(exe) + "\nIcon=" + escapeValue(icon) +
		"\nNoDisplay=true\nStartupWMClass=" + escapeValue(app.Name) + "\n"
	write := func() {
		if os.MkdirAll(filepath.Dir(entry), 0o755) == nil {
			_ = writeFile(entry, strings.NewReader(body), 0o644)
		}
	}
	if _, err := os.Stat(p.entry); err != nil {
		write()
	}
	standIn.Lock()
	standIn.write = write
	standIn.Unlock()
	return func() {
		standIn.Lock()
		standIn.write = nil
		standIn.Unlock()
		os.Remove(entry)
		os.RemoveAll(dir)
	}
}

// keepWindowEntry writes the open window's hidden entry, if there is a window.
func keepWindowEntry() {
	standIn.Lock()
	defer standIn.Unlock()
	if standIn.write != nil {
		standIn.write()
	}
}

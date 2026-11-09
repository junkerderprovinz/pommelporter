//go:build linux

package linux

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// WindowEntry gives the installer's window a hidden menu entry of its own,
// named by class, the window's WM_CLASS, and returns what removes it again.
// GNOME shows a window under the entry its class names and matches it only
// when it opens, so the window cannot stand under the app's own entry, which
// an uninstall takes away.
func (platform) WindowEntry(app setup.App, class string) (remove func()) {
	remove = func() {}
	p, err := locate(app)
	if err != nil || len(app.Icon) == 0 {
		return remove
	}
	dir, err := os.MkdirTemp("", "pommelporter-")
	if err != nil {
		return remove
	}
	icon := filepath.Join(dir, "icon.png")
	entry := strings.TrimSuffix(p.entry, ".desktop") + "-setup.desktop"
	// GLib ignores an entry whose Exec names no program it can find, and the
	// uninstaller that shows the window is gone before the window closes.
	// The entry is never started, so any program there is will do.
	body := "[Desktop Entry]\nType=Application\nName=" + escapeValue(app.Name) +
		"\nExec=true\nIcon=" + escapeValue(icon) +
		"\nNoDisplay=true\nStartupWMClass=" + escapeValue(class) + "\n"
	if os.MkdirAll(filepath.Dir(entry), 0o755) != nil ||
		writeFile(icon, bytes.NewReader(app.Icon), 0o644) != nil || writeFile(entry, strings.NewReader(body), 0o644) != nil {
		os.Remove(entry)
		os.RemoveAll(dir)
		return remove
	}
	return func() {
		os.Remove(entry)
		os.RemoveAll(dir)
	}
}

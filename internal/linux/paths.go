//go:build linux

package linux

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// paths are where one app's files go for the current user.
type paths struct {
	home    string
	dir     string // the program, its uninstaller and the state file
	exe     string
	entry   string // the menu entry
	icon    string
	hicolor string // the icon theme folder the icon is part of
	// link is the launcher in ~/.local/bin, or "" when that folder does not
	// exist and so is not on anybody's PATH.
	link string
	// desktop is the desktop copy of the entry, or "" without a desktop folder.
	desktop string
}

func locate(app setup.App) (paths, error) {
	home := os.Getenv("HOME")
	if !filepath.IsAbs(home) {
		return paths{}, errors.New("HOME does not name a folder to install into")
	}
	data := xdgDir("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	id := app.ID
	if id == "" {
		id = strings.ToLower(app.Name)
	}
	p := paths{
		home:    home,
		dir:     filepath.Join(data, id),
		entry:   filepath.Join(data, "applications", id+".desktop"),
		hicolor: filepath.Join(data, "icons", "hicolor"),
	}
	p.exe = filepath.Join(p.dir, app.Executable)
	p.icon = filepath.Join(p.hicolor, "256x256", "apps", id+".png")
	if bin := filepath.Join(home, ".local", "bin"); isDir(bin) {
		p.link = filepath.Join(bin, strings.ToLower(app.Executable))
	}
	if desk := desktopDir(home); desk != "" {
		p.desktop = filepath.Join(desk, id+".desktop")
	}
	return p, nil
}

// xdgDir reads a base directory variable. The XDG spec has a relative value
// ignored, the same as an empty one.
func xdgDir(env, fallback string) string {
	if dir := os.Getenv(env); filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	return fallback
}

// desktopDir finds the desktop folder the way xdg-user-dir does, asking that
// tool when it is installed. It returns "" when the folder does not exist or
// is the home folder, which is what xdg-user-dir names when there is no
// desktop.
func desktopDir(home string) string {
	var dir string
	if out, err := exec.Command("xdg-user-dir", "DESKTOP").Output(); err == nil {
		dir = strings.TrimSpace(string(out))
	} else {
		dir = userDir(home, "XDG_DESKTOP_DIR", filepath.Join(home, "Desktop"))
	}
	if !filepath.IsAbs(dir) || filepath.Clean(dir) == filepath.Clean(home) || !isDir(dir) {
		return ""
	}
	return dir
}

// userDir reads one entry of user-dirs.dirs, whose values are an absolute
// path or one starting at $HOME, in double quotes.
func userDir(home, key, fallback string) string {
	f, err := os.Open(filepath.Join(xdgDir("XDG_CONFIG_HOME", filepath.Join(home, ".config")), "user-dirs.dirs"))
	if err != nil {
		return fallback
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, value, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || name != key {
			continue
		}
		value = strings.Trim(value, `"`)
		if rest, ok := strings.CutPrefix(value, "$HOME"); ok {
			return home + rest
		}
		return value
	}
	return fallback
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

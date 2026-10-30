//go:build linux

// Package linux installs an app for the person running the installer, under
// ~/.local, with an entry in the desktop's application menu.
package linux

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const (
	// uninstaller is the installer's copy in the install folder; the menu
	// entry's Uninstall action runs it with --uninstall.
	uninstaller = "uninstall"
	// stateFile sits in the install folder, so uninstalling drops the
	// recorded choice together with the copy it describes.
	stateFile = "install.json"
)

type platform struct{}

// New returns the Linux platform.
func New() setup.Platform { return platform{} }

// state is what an install records for the next run of the installer.
type state struct {
	Version   string `json:"version"`
	StartMenu bool   `json:"startMenu"`
	Desktop   bool   `json:"desktop"`
	// Language is what the app first starts in; it reads it from here.
	Language string `json:"language,omitempty"`
}

func (platform) Defaults(app setup.App) setup.Options {
	o := setup.Options{StartMenu: true, Desktop: true}
	p, err := locate(app)
	if err != nil {
		return o
	}
	o.Dir = p.dir
	if s, ok := loadState(p); ok {
		o.StartMenu, o.Desktop, o.Language = s.StartMenu, s.Desktop, s.Language
	}
	return o
}

func (platform) Existing(app setup.App) *setup.Existing {
	p, err := locate(app)
	if err != nil {
		return nil
	}
	s, ok := loadState(p)
	if !ok {
		return nil
	}
	if _, err := os.Stat(p.exe); err != nil {
		return nil
	}
	return &setup.Existing{Version: s.Version, Dir: p.dir}
}

// CheckDir accepts the one folder an install for the current user goes to.
func (platform) CheckDir(app setup.App, dir string) setup.DirCheck {
	p, err := locate(app)
	if err != nil {
		return setup.DirCheck{Code: setup.DirFixed, Detail: err.Error()}
	}
	if filepath.Clean(dir) != p.dir {
		return setup.DirCheck{Code: setup.DirFixed, Detail: app.Name + " installs into " + p.dir}
	}
	return setup.DirCheck{OK: true}
}

func (platform) Roots() []string { return []string{"/"} }

func (platform) Install(ctx context.Context, app setup.App, o setup.Options, report setup.Report) error {
	p, err := locate(app)
	if err != nil {
		return err
	}
	report(setup.StepPrepare)
	if err := os.MkdirAll(p.dir, 0o755); err != nil {
		return err
	}

	report(setup.StepFiles)
	if err := copyPayload(app, p.dir); err != nil {
		return err
	}
	if err := copyInstaller(filepath.Join(p.dir, uninstaller)); err != nil {
		return err
	}

	report(setup.StepShortcuts)
	if err := placeShortcuts(ctx, app, o, p); err != nil {
		return err
	}

	report(setup.StepRegister)
	data, err := json.Marshal(state{Version: app.Version, StartMenu: o.StartMenu, Desktop: o.Desktop, Language: o.Language})
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(p.dir, stateFile), bytes.NewReader(data), 0o644)
}

// Uninstall removes the program, its entries and what its updater left in
// the install folder. Settings live under ~/.config and stay, and the window
// asks about the webview's storage on Windows only.
func (platform) Uninstall(ctx context.Context, app setup.App, _ setup.Removal, report setup.Report) error {
	p, err := locate(app)
	if err != nil {
		return err
	}
	report(setup.StepRemove)
	var errs []error
	if p.desktop != "" {
		errs = append(errs, removeFile(p.desktop))
	}
	if p.link != "" {
		if target, err := os.Readlink(p.link); err == nil && target == p.exe {
			errs = append(errs, removeFile(p.link))
		}
	}
	errs = append(errs, removeFile(p.entry), removeFile(p.icon), os.RemoveAll(p.dir))
	refreshCaches(ctx, p)
	return errors.Join(errs...)
}

// Launch starts the program in a session of its own, so it outlives the
// installer and the terminal the installer may have been started from.
func (platform) Launch(app setup.App) error {
	p, err := locate(app)
	if err != nil {
		return err
	}
	cmd := exec.Command(p.exe)
	cmd.Dir = p.home
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func loadState(p paths) (state, bool) {
	var s state
	data, err := os.ReadFile(filepath.Join(p.dir, stateFile))
	if err != nil || json.Unmarshal(data, &s) != nil {
		return s, false
	}
	return s, true
}

// copyPayload writes every file of the payload into dir. The program and
// anything the payload marks executable get the execute bits, which an
// embedded file system does not carry.
func copyPayload(app setup.App, dir string) error {
	return fs.WalkDir(app.Payload, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		perm := fs.FileMode(0o644)
		if name == app.Executable || info.Mode()&0o111 != 0 {
			perm = 0o755
		}
		f, err := app.Payload.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		return writeFile(dest, f, perm)
	})
}

func copyInstaller(dest string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	f, err := os.Open(self)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeFile(dest, f, 0o755)
}

// placeShortcuts writes the icon, the menu entry, the desktop copy and the
// launcher in ~/.local/bin. An entry left out on a reinstall is removed.
func placeShortcuts(ctx context.Context, app setup.App, o setup.Options, p paths) error {
	if len(app.Icon) > 0 {
		if err := os.MkdirAll(filepath.Dir(p.icon), 0o755); err != nil {
			return err
		}
		if err := writeFile(p.icon, bytes.NewReader(app.Icon), 0o644); err != nil {
			return err
		}
	}
	entry := desktopEntry(app, p)
	if o.StartMenu {
		if err := os.MkdirAll(filepath.Dir(p.entry), 0o755); err != nil {
			return err
		}
		if err := writeFile(p.entry, strings.NewReader(entry), 0o644); err != nil {
			return err
		}
	} else if err := removeFile(p.entry); err != nil {
		return err
	}
	if p.desktop != "" {
		if o.Desktop {
			// GNOME and Xfce start a launcher on the desktop only when it is
			// executable and, for GNOME, marked trusted.
			if err := writeFile(p.desktop, strings.NewReader(entry), 0o755); err != nil {
				return err
			}
			run(ctx, "gio", "set", p.desktop, "metadata::trusted", "true")
		} else if err := removeFile(p.desktop); err != nil {
			return err
		}
	}
	if p.link != "" {
		if err := linkLauncher(p.link, p.exe); err != nil {
			return err
		}
	}
	refreshCaches(ctx, p)
	return nil
}

// linkLauncher points link at the program unless the name is taken, by an
// earlier install of this app or by another program that keeps it.
func linkLauncher(link, exe string) error {
	if _, err := os.Lstat(link); err == nil {
		return nil
	}
	return os.Symlink(exe, link)
}

// refreshCaches tells the menu and the icon theme about the change. Both
// tools are optional; desktops without them watch the folders instead.
func refreshCaches(ctx context.Context, p paths) {
	run(ctx, "update-desktop-database", "-q", filepath.Dir(p.entry))
	if _, err := os.Stat(p.hicolor); err == nil {
		run(ctx, "gtk-update-icon-cache", "-q", "-t", "-f", p.hicolor)
	}
}

// run starts a helper when it is installed and ignores how it ends, since
// nothing it does is needed for the program to work.
func run(ctx context.Context, name string, args ...string) {
	path, err := exec.LookPath(name)
	if err != nil {
		return
	}
	exec.CommandContext(ctx, path, args...).Run()
}

// writeFile replaces name through a temporary file and a rename, so a copy
// of the program that is running keeps its file and nobody sees half of the
// new one.
func writeFile(name string, r io.Reader, perm fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".install-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}

func removeFile(name string) error {
	if err := os.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

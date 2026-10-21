//go:build windows

// Package windows installs an app for all users under Program Files or a
// folder only administrators can change.
package windows

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const uninstaller = "uninstall.exe"

// places are the folders the installer writes to besides the install folder.
type places struct {
	programFiles   string
	programData    string
	commonPrograms string // the Start menu of all users
	publicDesktop  string
	userPrograms   string // the Start menu of the person running the installer
	userDesktop    string
	appData        string // their roaming AppData, where the webview keeps its cache
}

type platform struct {
	// machine and user stand for HKLM and HKCU. The installer is a 64-bit
	// program, so they are the 64-bit views.
	machine  registry.Key
	user     registry.Key
	places   places
	schtasks func(args ...string) error
}

// New returns the Windows platform.
func New() setup.Platform {
	return &platform{
		machine: registry.LOCAL_MACHINE,
		user:    registry.CURRENT_USER,
		places: places{
			programFiles:   known(windows.FOLDERID_ProgramFiles),
			programData:    known(windows.FOLDERID_ProgramData),
			commonPrograms: known(windows.FOLDERID_CommonPrograms),
			publicDesktop:  known(windows.FOLDERID_PublicDesktop),
			userPrograms:   known(windows.FOLDERID_Programs),
			userDesktop:    known(windows.FOLDERID_Desktop),
			appData:        known(windows.FOLDERID_RoamingAppData),
		},
		schtasks: runSchtasks,
	}
}

// known returns a folder every Windows installation has.
func known(id *windows.KNOWNFOLDERID) string {
	dir, _ := windows.KnownFolderPath(id, 0)
	return dir
}

func entryKey(app setup.App) string  { return uninstallRoot + app.Name }
func choiceKey(app setup.App) string { return `Software\` + app.Name }

func (p *platform) Defaults(app setup.App) setup.Options {
	dir := installDirOf(p.machine, entryKey(app))
	if dir == "" {
		dir = filepath.Join(p.places.programFiles, app.Name)
	}
	return setup.Options{
		Dir:       dir,
		StartMenu: p.choice(app, "StartMenuShortcut"),
		Desktop:   p.choice(app, "DesktopShortcut"),
	}
}

// choice reads the last choice for a shortcut, else the one an earlier
// installer kept for this person; a first install has both.
func (p *platform) choice(app setup.App, name string) bool {
	if v, ok := readDWORD(p.machine, choiceKey(app), name); ok {
		return v != 0
	}
	for _, key := range app.Legacy.ChoiceKeys {
		if v, ok := readDWORD(p.user, key, name); ok {
			return v != 0
		}
	}
	return true
}

func (p *platform) Existing(app setup.App) *setup.Existing {
	for _, e := range []struct {
		root registry.Key
		keys []string
	}{
		{p.machine, []string{entryKey(app)}},
		{p.machine, app.Legacy.MachineKeys},
		{p.user, app.Legacy.UserKeys},
	} {
		for _, key := range e.keys {
			if !keyExists(e.root, key) {
				continue
			}
			return &setup.Existing{Version: readString(e.root, key, "DisplayVersion"), Dir: installDirOf(e.root, key)}
		}
	}
	return nil
}

func (p *platform) CheckDir(app setup.App, dir string) setup.DirCheck {
	if !filepath.IsAbs(dir) {
		return refuse(setup.DirRelative, "%s is not a full path", dir)
	}
	norm, err := normalize(dir)
	if err != nil {
		return refuse(setup.DirRelative, "%s is not a usable path: %v", dir, err)
	}
	return checkDir(norm, func(d string) bool { return p.holdsEarlier(app, d) })
}

// holdsEarlier reports whether dir is where this app is installed, or holds
// an earlier installation for all users whose uninstaller clears it.
func (p *platform) holdsEarlier(app setup.App, dir string) bool {
	if loc := installDirOf(p.machine, entryKey(app)); loc != "" && strings.EqualFold(loc, dir) {
		return true
	}
	for _, key := range app.Legacy.MachineKeys {
		loc := installDirOf(p.machine, key)
		if loc != "" && (strings.EqualFold(loc, dir) || strings.HasPrefix(strings.ToLower(loc), strings.ToLower(dir)+`\`)) {
			return true
		}
	}
	return false
}

func (p *platform) Roots() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var roots []string
	for i := range 26 {
		if mask&(1<<i) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		if windows.GetDriveType(windows.StringToUTF16Ptr(root)) == windows.DRIVE_FIXED {
			roots = append(roots, root)
		}
	}
	return roots
}

func (p *platform) Install(ctx context.Context, app setup.App, o setup.Options, report setup.Report) error {
	report(setup.StepPrepare)
	if chk := p.CheckDir(app, o.Dir); !chk.OK {
		return errors.New(chk.Detail)
	}
	dir, err := normalize(o.Dir)
	if err != nil {
		return err
	}
	exe := filepath.Join(dir, app.Executable)
	earlier := p.holdsEarlier(app, dir)

	report(setup.StepClose)
	closeRunning(dir, app.InstanceID)

	report(setup.StepLegacy)
	if err := p.removeMachineCopies(ctx, app); err != nil {
		return err
	}
	p.removeUserCopies(app, exe)

	report(setup.StepFiles)
	size, err := placeFiles(app, dir, earlier)
	if err != nil {
		return err
	}

	report(setup.StepShortcuts)
	if err := p.shortcuts(app, exe, o); err != nil {
		return err
	}

	report(setup.StepRegister)
	if err := p.register(app, dir, size, o); err != nil {
		return err
	}

	// The program works without the task, it only stays at this version, so
	// a folder that cannot be made safe does not fail the install. A task
	// from an earlier install would use that folder all the same and goes.
	report(setup.StepUpdates)
	if err := prepareData(filepath.Join(p.places.programData, app.Name)); err != nil {
		p.deleteTask(app)
		return nil
	}
	p.createTask(app, exe)
	return nil
}

// placeFiles makes dir safe, then puts the payload and the uninstaller in it
// and returns their size. A folder that held nothing of the app's must still
// be empty once nobody else can change it.
func placeFiles(app setup.App, dir string, earlier bool) (int64, error) {
	if err := lockChain(dir); err != nil {
		return 0, err
	}
	if !earlier {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return 0, err
		}
		if len(entries) > 0 {
			return 0, fmt.Errorf("%s already holds other files", dir)
		}
	}
	if err := stepAside(filepath.Join(dir, app.Executable)); err != nil {
		return 0, err
	}

	var size int64
	err := fs.WalkDir(app.Payload, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || name == "." {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if d.IsDir() {
			if err := os.Mkdir(target, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
			return nil
		}
		f, err := app.Payload.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		size += info.Size()
		return placeFile(target, f)
	})
	if err != nil {
		return 0, err
	}

	self, err := os.Executable()
	if err != nil {
		return 0, err
	}
	un := filepath.Join(dir, uninstaller)
	if !strings.EqualFold(filepath.Clean(self), un) {
		f, err := os.Open(self)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		if err := placeFile(un, f); err != nil {
			return 0, err
		}
	}
	if info, err := os.Stat(un); err == nil {
		size += info.Size()
	}
	return size, nil
}

// placeFile writes a new file and moves it over target, so the file carries
// the folder's rights rather than those of a file it replaces.
func placeFile(target string, r io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".install-*")
	if err != nil {
		return err
	}
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// shortcuts creates the chosen ones and removes the others, so an update
// does not bring back a shortcut somebody left out.
func (p *platform) shortcuts(app setup.App, exe string, o setup.Options) error {
	for _, s := range []struct {
		dir  string
		want bool
	}{
		{p.places.commonPrograms, o.StartMenu},
		{p.places.publicDesktop, o.Desktop},
	} {
		lnk := filepath.Join(s.dir, app.Name+".lnk")
		if !s.want {
			if err := os.Remove(lnk); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		if err := makeShortcut(lnk, exe); err != nil {
			return fmt.Errorf("creating %s: %w", lnk, err)
		}
	}
	return nil
}

// register writes the entry under Apps and the shortcut choice. The apps
// compare their own folder with InstallLocation to tell the installed copy
// from a portable one, and their updater keeps DisplayVersion current while
// DisplayIcon names the program.
func (p *platform) register(app setup.App, dir string, size int64, o setup.Options) error {
	k, _, err := registry.CreateKey(p.machine, entryKey(app), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	un := `"` + filepath.Join(dir, uninstaller) + `" --uninstall`
	for name, v := range map[string]string{
		"DisplayName":          app.Name,
		"DisplayVersion":       app.Version,
		"Publisher":            app.Publisher,
		"DisplayIcon":          filepath.Join(dir, app.Executable),
		"InstallLocation":      dir,
		"UninstallString":      un,
		"QuietUninstallString": un + " /S",
		"URLInfoAbout":         app.Homepage,
	} {
		if err := k.SetStringValue(name, v); err != nil {
			return err
		}
	}
	for name, v := range map[string]uint32{
		"EstimatedSize": uint32((size + 1023) / 1024),
		"NoModify":      1,
		"NoRepair":      1,
	} {
		if err := k.SetDWordValue(name, v); err != nil {
			return err
		}
	}

	c, _, err := registry.CreateKey(p.machine, choiceKey(app), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.SetDWordValue("StartMenuShortcut", flag(o.StartMenu)); err != nil {
		return err
	}
	return c.SetDWordValue("DesktopShortcut", flag(o.Desktop))
}

func flag(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// installedDir is the folder of the installed copy, or that of the running
// uninstall.exe when the entry is gone.
func (p *platform) installedDir(app setup.App) (string, error) {
	if dir := installDirOf(p.machine, entryKey(app)); dir != "" {
		return dir, nil
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Base(self), uninstaller) {
		return "", fmt.Errorf("%s is not installed", app.Name)
	}
	return filepath.Dir(self), nil
}

func (p *platform) Uninstall(ctx context.Context, app setup.App, report setup.Report) error {
	dir, err := p.installedDir(app)
	if err != nil {
		return err
	}
	exe := filepath.Join(dir, app.Executable)

	report(setup.StepClose)
	closeRunning(dir, app.InstanceID)

	report(setup.StepRemove)
	p.deleteTask(app)
	var errs []error
	keep := func(err error) {
		if err != nil {
			errs = append(errs, err)
		}
	}
	keep(removeTree(filepath.Join(p.places.programData, app.Name)))
	// The webview's cache of the person uninstalling; their settings and
	// downloads stay.
	keep(removeTree(filepath.Join(p.places.appData, app.Executable)))

	// Only what the installer and the updates put there, since the folder
	// could have been chosen by hand. A copy that still runs goes at the
	// next start, and so does the uninstaller running now.
	var dirs []string
	fs.WalkDir(app.Payload, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || name == "." {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if d.IsDir() {
			dirs = append(dirs, target)
			return nil
		}
		keep(removeSoon(target))
		return nil
	})
	keep(removeSoon(exe + ".old"))
	keep(removeMatches(filepath.Join(dir, "."+app.Executable+".update-*")))
	keep(removeSoon(filepath.Join(dir, uninstaller)))
	slices.Reverse(dirs)
	for _, d := range dirs {
		os.Remove(d)
	}
	keep(removeDirSoon(dir))

	for _, lnk := range []string{
		filepath.Join(p.places.commonPrograms, app.Name+".lnk"),
		filepath.Join(p.places.publicDesktop, app.Name+".lnk"),
	} {
		if err := os.Remove(lnk); err != nil && !errors.Is(err, fs.ErrNotExist) {
			keep(err)
		}
	}
	keep(deleteKey(p.machine, choiceKey(app)))
	keep(deleteKey(p.machine, entryKey(app)))
	return errors.Join(errs...)
}

// Launch starts the installed program through Explorer, as the person signed
// in rather than with the installer's rights.
func (p *platform) Launch(app setup.App) error {
	dir, err := p.installedDir(app)
	if err != nil {
		return err
	}
	return shellExecute(filepath.Join(dir, app.Executable))
}

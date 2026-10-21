//go:build windows

package windows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

func appFixture() setup.App {
	return setup.App{
		Name:       "Prog",
		Executable: "Prog.exe",
		Version:    "2.1.0",
		Publisher:  "Tom & Jerry",
		Homepage:   "https://example.com/prog",
		Autostart:  "Prog",
		Payload:    fstest.MapFS{"Prog.exe": {Data: []byte("program")}},
		Legacy: setup.Legacy{
			MachineKeys: []string{uninstallRoot + "ProgProg"},
			UserKeys:    []string{uninstallRoot + "ProgProg"},
			ChoiceKeys:  []string{`Software\ProgProg`},
		},
	}
}

func TestDefaultsOnAFirstInstall(t *testing.T) {
	p, _ := testPlatform(t)
	o := p.Defaults(appFixture())
	want := setup.Options{Dir: filepath.Join(p.places.programFiles, "Prog"), StartMenu: true, Desktop: true}
	if o != want {
		t.Errorf("Defaults = %+v, want %+v", o, want)
	}
}

func TestDefaultsFollowTheEarlierChoice(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	k, _, err := registry.CreateKey(p.user, app.Legacy.ChoiceKeys[0], registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	k.SetDWordValue("DesktopShortcut", 0)
	k.Close()
	if o := p.Defaults(app); !o.StartMenu || o.Desktop {
		t.Errorf("an earlier installer's choice: %+v", o)
	}

	if err := p.register(app, `D:\Apps\Prog`, 0, setup.Options{StartMenu: false, Desktop: true}); err != nil {
		t.Fatal(err)
	}
	o := p.Defaults(app)
	if o.Dir != `D:\Apps\Prog` || o.StartMenu || !o.Desktop {
		t.Errorf("the last choice: %+v", o)
	}
}

func TestRegisterWritesTheAppsEntry(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	if err := p.register(app, `D:\Apps\Prog`, 5000, setup.Options{StartMenu: true}); err != nil {
		t.Fatal(err)
	}
	key := entryKey(app)
	for name, want := range map[string]string{
		"DisplayName":          "Prog",
		"DisplayVersion":       "2.1.0",
		"Publisher":            "Tom & Jerry",
		"DisplayIcon":          `D:\Apps\Prog\Prog.exe`,
		"InstallLocation":      `D:\Apps\Prog`,
		"UninstallString":      `"D:\Apps\Prog\uninstall.exe" --uninstall`,
		"QuietUninstallString": `"D:\Apps\Prog\uninstall.exe" --uninstall /S`,
		"URLInfoAbout":         "https://example.com/prog",
	} {
		if got := readString(p.machine, key, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	for name, want := range map[string]uint64{"EstimatedSize": 5, "NoModify": 1, "NoRepair": 1} {
		if got, _ := readDWORD(p.machine, key, name); got != want {
			t.Errorf("%s = %d, want %d", name, got, want)
		}
	}
	if v, _ := readDWORD(p.machine, choiceKey(app), "DesktopShortcut"); v != 0 {
		t.Errorf("DesktopShortcut = %d", v)
	}
	if e := p.Existing(app); e == nil || e.Version != "2.1.0" || e.Dir != `D:\Apps\Prog` {
		t.Errorf("Existing = %+v", e)
	}
}

func TestExistingFindsAnOlderInstall(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	if p.Existing(app) != nil {
		t.Fatal("Existing found something in an empty registry")
	}
	setString(t, p.machine, app.Legacy.MachineKeys[0], "DisplayIcon", `C:\Program Files\Prog\Prog\Prog.exe`)
	setString(t, p.machine, app.Legacy.MachineKeys[0], "DisplayVersion", "1.1.0")
	if e := p.Existing(app); e == nil || e.Version != "1.1.0" || e.Dir != `C:\Program Files\Prog\Prog` {
		t.Errorf("Existing = %+v", e)
	}
}

func TestIconPath(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Prog\Prog.exe`:     `C:\Prog\Prog.exe`,
		`"C:\Prog\Prog.exe"`:   `C:\Prog\Prog.exe`,
		`"C:\Prog\Prog.exe",0`: `C:\Prog\Prog.exe`,
		`C:\Prog,v2\Prog.exe`:  `C:\Prog,v2\Prog.exe`,
		``:                     ``,
	} {
		if got := iconPath(in); got != want {
			t.Errorf("iconPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstToken(t *testing.T) {
	for in, want := range map[string]string{
		`"C:\Program Files\Prog\uninstall.exe"`:    `C:\Program Files\Prog\uninstall.exe`,
		`"C:\Program Files\Prog\uninstall.exe" /x`: `C:\Program Files\Prog\uninstall.exe`,
		`C:\Prog\uninstall.exe /x`:                 `C:\Prog\uninstall.exe`,
	} {
		if got := firstToken(in); got != want {
			t.Errorf("firstToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDeleteKeyRemovesTheWholeKey(t *testing.T) {
	root, _ := testKey(t)
	setString(t, root, `Prog\Sub`, "v", "x")
	if err := deleteKey(root, "Prog"); err != nil {
		t.Fatal(err)
	}
	if keyExists(root, "Prog") {
		t.Error("the key is still there")
	}
	if err := deleteKey(root, "Prog"); err != nil {
		t.Errorf("deleting a missing key: %v", err)
	}
}

func TestOlderMachineInstallIsUninstalledFirst(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	key := app.Legacy.MachineKeys[0]
	_, regPath := testKey(t)
	// Point the platform at the key reg.exe can name, and let the "uninstaller"
	// remove its own entry the way an NSIS uninstaller does.
	root, err := registry.OpenKey(registry.CURRENT_USER, strings.TrimPrefix(regPath, `HKCU\`), registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	p.machine = root
	setString(t, root, key, "UninstallString",
		`"`+os.Getenv("SystemRoot")+`\System32\cmd.exe" /c reg delete "`+regPath+`\`+key+`" /f >nul & rem`)

	start := time.Now()
	if err := p.removeMachineCopies(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	if keyExists(root, key) {
		t.Error("the older entry is still there")
	}
	if time.Since(start) > legacyWait/2 {
		t.Error("waited although the entry was gone")
	}
}

func TestOlderUserInstallIsRemoved(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	old := filepath.Join(t.TempDir(), "Programs", "Prog", "Prog.exe")
	dir := filepath.Dir(old)
	for _, f := range []string{old, old + ".old", filepath.Join(dir, "uninstall.exe"), filepath.Join(dir, ".Prog.exe.update-123")} {
		writeFile(t, f, "x")
	}
	for _, f := range []string{filepath.Join(p.places.userPrograms, "Prog.lnk"), filepath.Join(p.places.userDesktop, "Prog.lnk")} {
		writeFile(t, f, "x")
	}
	key := app.Legacy.UserKeys[0]
	setString(t, p.user, key, "DisplayIcon", old)
	setString(t, p.user, app.Legacy.ChoiceKeys[0], "x", "x")
	setString(t, p.user, runKey, "Prog", `"`+strings.ToUpper(old)+`"`)
	setString(t, p.user, runKey, "Other", `"`+old+`"`)

	exe := `C:\Program Files\Prog\Prog.exe`
	p.removeUserCopies(app, exe)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(dir)
		t.Errorf("the old folder is still there with %d entries", len(entries))
	}
	for _, f := range []string{filepath.Join(p.places.userPrograms, "Prog.lnk"), filepath.Join(p.places.userDesktop, "Prog.lnk")} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("%s is still there", f)
		}
	}
	if keyExists(p.user, key) || keyExists(p.user, app.Legacy.ChoiceKeys[0]) {
		t.Error("the older keys are still there")
	}
	if got := readString(p.user, runKey, "Prog"); got != `"`+exe+`"` {
		t.Errorf("autostart = %s", got)
	}
	if got := readString(p.user, runKey, "Other"); got != `"`+old+`"` {
		t.Errorf("another Run value changed to %s", got)
	}
}

func TestOlderUserInstallBehindAJunctionKeepsItsTarget(t *testing.T) {
	p, _ := testPlatform(t)
	app := appFixture()
	tmp := t.TempDir()
	elsewhere := filepath.Join(tmp, "elsewhere")
	writeFile(t, filepath.Join(elsewhere, "uninstall.exe"), "keep")
	link := filepath.Join(tmp, "Prog")
	junction(t, link, elsewhere)
	setString(t, p.user, app.Legacy.UserKeys[0], "DisplayIcon", filepath.Join(link, "Prog.exe"))

	p.removeUserCopies(app, `C:\Program Files\Prog\Prog.exe`)
	if _, err := os.Stat(filepath.Join(elsewhere, "uninstall.exe")); err != nil {
		t.Errorf("a file behind the junction went: %v", err)
	}
}

func TestTaskDefinition(t *testing.T) {
	b := taskXML(appFixture(), `C:\Program Files\Prog\Prog.exe`)
	if b[0] != 0xff || b[1] != 0xfe {
		t.Fatalf("no UTF-16 byte order mark: % x", b[:2])
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	text := string(utf16.Decode(u[1:]))
	for _, want := range []string{
		`<Command>"C:\Program Files\Prog\Prog.exe"</Command>`,
		`<Arguments>--update</Arguments>`,
		`<UserId>S-1-5-18</UserId>`,
		`<Author>Tom &amp; Jerry</Author>`,
		`<Delay>PT5M</Delay>`,
		`<RandomDelay>PT1H</RandomDelay>`,
		"\r\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the task lacks %q", want)
		}
	}
}

func TestRemoveTreeLeavesJunctionTargets(t *testing.T) {
	tmp := t.TempDir()
	outside := filepath.Join(tmp, "outside")
	writeFile(t, filepath.Join(outside, "keep.txt"), "x")
	tree := filepath.Join(tmp, "tree")
	writeFile(t, filepath.Join(tree, "sub", "file"), "x")
	junction(t, filepath.Join(tree, "sub", "link"), outside)
	junction(t, filepath.Join(tmp, "rootlink"), outside)

	if err := removeTree(tree); err != nil {
		t.Fatal(err)
	}
	if err := removeTree(filepath.Join(tmp, "rootlink")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Error("the tree is still there")
	}
	if _, err := os.Stat(filepath.Join(outside, "keep.txt")); err != nil {
		t.Errorf("removeTree followed a junction: %v", err)
	}
	if err := removeTree(filepath.Join(tmp, "missing")); err != nil {
		t.Errorf("a missing path: %v", err)
	}
}

func TestStepAsideRemovesAProgramNotRunning(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "Prog.exe")
	writeFile(t, exe, "x")
	writeFile(t, exe+".old", "x")
	if err := stepAside(exe); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("%d files left", len(entries))
	}
}

// running copies a system program to path and keeps it running until the
// test ends.
func running(t *testing.T, path string) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(os.Getenv("SystemRoot"), "System32", "ping.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(path, "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
}

func TestStepAsideRenamesARunningProgram(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "Prog.exe")
	running(t, exe)
	if err := stepAside(exe); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(exe + ".old"); err != nil {
		t.Errorf("no .old: %v", err)
	}

	running(t, exe)
	if err := stepAside(exe); err != nil {
		t.Fatal(err)
	}
	if aside, _ := filepath.Glob(filepath.Join(dir, ".Prog.exe.update-*")); len(aside) != 1 {
		t.Errorf("with .old held, the program went to %v", aside)
	}
	if _, err := os.Stat(exe); !os.IsNotExist(err) {
		t.Error("the program is still in place")
	}
}

func TestCloseRunningWithoutALock(t *testing.T) {
	if closeRunning(t.TempDir(), "") || closeRunning(t.TempDir(), randomName("pommelporter-test-")) {
		t.Error("closed something that does not run")
	}
}

func TestMakeShortcut(t *testing.T) {
	lnk := filepath.Join(t.TempDir(), "Prog.lnk")
	if err := makeShortcut(lnk, filepath.Join(os.Getenv("SystemRoot"), "notepad.exe")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(lnk)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 4 || b[0] != 0x4c {
		t.Errorf("not a shell link: % x", b[:min(4, len(b))])
	}
}

func TestCheckArch(t *testing.T) {
	app := appFixture()
	native := nativeMachine(t)
	app.Payload = fstest.MapFS{"Prog.exe": {Data: fakeProgram(native)}}
	if err := checkArch(app); err != nil {
		t.Errorf("a program for this machine: %v", err)
	}
	other := uint16(machineARM64)
	if native == machineARM64 {
		other = machineAMD64
	}
	app.Payload = fstest.MapFS{"Prog.exe": {Data: fakeProgram(other)}}
	if err := checkArch(app); err == nil {
		t.Error("accepted a program for another processor")
	}
	app.Payload = fstest.MapFS{"Prog.exe": {Data: []byte("#!/bin/sh")}}
	if err := checkArch(app); err == nil {
		t.Error("accepted something that is not a program")
	}
}

func TestReadAutoUpdate(t *testing.T) {
	dir := t.TempDir()
	if !readAutoUpdate(filepath.Join(dir, "missing.json")) {
		t.Error("a missing file turned updates off")
	}
	off := filepath.Join(dir, "off.json")
	writeFile(t, off, `{"autoUpdate":false}`)
	if readAutoUpdate(off) {
		t.Error("false was not kept")
	}
}

func TestInstallAndUninstall(t *testing.T) {
	needAdmin(t)
	p, tasks := testPlatform(t)
	app := appFixture()
	native := nativeMachine(t)
	app.Payload = fstest.MapFS{
		"Prog.exe":       {Data: fakeProgram(native)},
		"res/strings.js": {Data: []byte("x")},
	}
	dir := filepath.Join(safeBase(t), "Prog")
	var steps []string
	report := func(s string) { steps = append(steps, s) }

	var task string
	p.schtasks = func(args ...string) error {
		for i, a := range args {
			if a == "/XML" {
				b, _ := os.ReadFile(args[i+1])
				task = string(b)
			}
		}
		return tasks.run(args...)
	}
	o := setup.Options{Dir: dir, StartMenu: true, Desktop: false}
	if err := p.Install(context.Background(), app, o, report); err != nil {
		t.Fatal(err)
	}
	want := []string{setup.StepPrepare, setup.StepClose, setup.StepLegacy, setup.StepFiles, setup.StepShortcuts, setup.StepRegister, setup.StepUpdates}
	if strings.Join(steps, ",") != strings.Join(want, ",") {
		t.Errorf("steps = %v", steps)
	}
	for _, f := range []string{"Prog.exe", `res\strings.js`, "uninstall.exe"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	lockedOK(t, dir)
	if got := readString(p.machine, entryKey(app), "InstallLocation"); got != dir {
		t.Errorf("InstallLocation = %s", got)
	}
	if _, err := os.Stat(filepath.Join(p.places.commonPrograms, "Prog.lnk")); err != nil {
		t.Errorf("Start menu entry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.places.publicDesktop, "Prog.lnk")); !os.IsNotExist(err) {
		t.Error("a desktop shortcut nobody chose")
	}
	data := filepath.Join(p.places.programData, "Prog")
	lockedOK(t, data)
	if b, _ := os.ReadFile(filepath.Join(data, settingsFile)); !strings.Contains(string(b), `"autoUpdate":true`) {
		t.Errorf("settings.json = %q", b)
	}
	if task == "" {
		t.Error("no task was registered")
	}

	steps = nil
	if err := p.Uninstall(context.Background(), app, report); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{dir, data, filepath.Join(p.places.commonPrograms, "Prog.lnk")} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s is still there", gone)
		}
	}
	if keyExists(p.machine, entryKey(app)) || keyExists(p.machine, choiceKey(app)) {
		t.Error("the keys are still there")
	}
	last := tasks.calls[len(tasks.calls)-1]
	if last[0] != "/Delete" {
		t.Errorf("the task was not deleted: %v", last)
	}
}

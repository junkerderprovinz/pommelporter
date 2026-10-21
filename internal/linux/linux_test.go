//go:build linux

package linux

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const testID = "com.example.knightloader"

func testApp() setup.App {
	return setup.App{
		Name:       "KnightLoader",
		ID:         testID,
		Executable: "knightloader",
		Version:    "1.2.0",
		Icon:       []byte("png"),
		Categories: []string{"Network", "FileTransfer"},
		Payload: fstest.MapFS{
			// The redirection is the shell's own, so the script runs without
			// a PATH.
			"knightloader":     {Data: []byte("#!/bin/sh\n: > \"$HOME/launched\"\n"), Mode: 0o444},
			"share/readme.txt": {Data: []byte("hello"), Mode: 0o444},
		},
	}
}

// newHome gives the test a home folder with a desktop and ~/.local/bin, and a
// PATH on which none of the optional helpers is found.
func newHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir())
	for _, dir := range []string{"Desktop", ".local/bin"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func install(t *testing.T, app setup.App, o setup.Options) []string {
	t.Helper()
	var steps []string
	if err := New().Install(context.Background(), app, o, func(s string) { steps = append(steps, s) }); err != nil {
		t.Fatalf("Install: %v", err)
	}
	return steps
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestInstallPutsTheProgramAndItsEntriesInPlace(t *testing.T) {
	home := newHome(t)
	app := testApp()
	o := New().Defaults(app)
	steps := install(t, app, o)

	want := []string{setup.StepPrepare, setup.StepFiles, setup.StepShortcuts, setup.StepRegister}
	if !slices.Equal(steps, want) {
		t.Errorf("steps = %v, want %v", steps, want)
	}
	dir := filepath.Join(home, ".local/share", testID)
	if o.Dir != dir {
		t.Errorf("Defaults().Dir = %q, want %q", o.Dir, dir)
	}
	exe := filepath.Join(dir, "knightloader")
	if m := mode(t, exe); m != 0o755 {
		t.Errorf("program mode = %v, want 0755", m)
	}
	if m := mode(t, filepath.Join(dir, "share/readme.txt")); m != 0o644 {
		t.Errorf("payload file mode = %v, want 0644", m)
	}
	if m := mode(t, filepath.Join(dir, "uninstall")); m != 0o755 {
		t.Errorf("uninstaller mode = %v, want 0755", m)
	}

	entry, err := os.ReadFile(filepath.Join(home, ".local/share/applications", testID+".desktop"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"Name=KnightLoader",
		"Exec=" + exe,
		"Icon=" + testID,
		"Categories=Network;FileTransfer;",
		"StartupWMClass=KnightLoader",
		"Exec=" + filepath.Join(dir, "uninstall") + " --uninstall",
	} {
		if !strings.Contains(string(entry), line+"\n") {
			t.Errorf("menu entry lacks %q:\n%s", line, entry)
		}
	}
	if !exists(filepath.Join(home, ".local/share/icons/hicolor/256x256/apps", testID+".png")) {
		t.Error("no icon in the hicolor theme")
	}
	if m := mode(t, filepath.Join(home, "Desktop", testID+".desktop")); m != 0o755 {
		t.Errorf("desktop copy mode = %v, want 0755", m)
	}
	if target, err := os.Readlink(filepath.Join(home, ".local/bin/knightloader")); err != nil || target != exe {
		t.Errorf("launcher points at %q (%v), want %q", target, err, exe)
	}

	got := New().Existing(app)
	if got == nil || got.Version != "1.2.0" || got.Dir != dir {
		t.Errorf("Existing = %+v, want version 1.2.0 in %s", got, dir)
	}
}

func TestNothingIsInstalledBeforeTheFirstInstall(t *testing.T) {
	newHome(t)
	app := testApp()
	if got := New().Existing(app); got != nil {
		t.Errorf("Existing = %+v, want nil", got)
	}
	if o := New().Defaults(app); !o.StartMenu || !o.Desktop {
		t.Errorf("Defaults = %+v, want both entries on", o)
	}
}

func TestReinstallKeepsTheChoiceAndRemovesEntriesLeftOut(t *testing.T) {
	home := newHome(t)
	app := testApp()
	install(t, app, New().Defaults(app))

	o := New().Defaults(app)
	o.StartMenu, o.Desktop = false, false
	install(t, app, o)

	for _, path := range []string{
		filepath.Join(home, ".local/share/applications", testID+".desktop"),
		filepath.Join(home, "Desktop", testID+".desktop"),
	} {
		if exists(path) {
			t.Errorf("%s is still there", path)
		}
	}
	if d := New().Defaults(app); d.StartMenu || d.Desktop {
		t.Errorf("Defaults = %+v, want the last choice with both entries off", d)
	}
}

func TestUninstallRemovesEverythingButTheSettings(t *testing.T) {
	home := newHome(t)
	app := testApp()
	install(t, app, New().Defaults(app))
	dir := filepath.Join(home, ".local/share", testID)
	// What the app's updater leaves next to the program.
	for _, name := range []string{"knightloader.old", ".knightloader.update-123"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	settings := filepath.Join(home, ".config/KnightLoader/settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var steps []string
	if err := New().Uninstall(context.Background(), app, func(s string) { steps = append(steps, s) }); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !slices.Equal(steps, []string{setup.StepRemove}) {
		t.Errorf("steps = %v", steps)
	}
	for _, path := range []string{
		dir,
		filepath.Join(home, ".local/share/applications", testID+".desktop"),
		filepath.Join(home, ".local/share/icons/hicolor/256x256/apps", testID+".png"),
		filepath.Join(home, "Desktop", testID+".desktop"),
		filepath.Join(home, ".local/bin/knightloader"),
	} {
		if exists(path) {
			t.Errorf("%s is still there", path)
		}
	}
	if !exists(settings) {
		t.Error("the settings went with the program")
	}
	if New().Existing(app) != nil {
		t.Error("Existing still reports a copy")
	}
}

func TestLauncherLeavesAnotherProgramsNameAlone(t *testing.T) {
	home := newHome(t)
	app := testApp()
	other := filepath.Join(home, ".local/bin/knightloader")
	if err := os.WriteFile(other, []byte("someone else's"), 0o755); err != nil {
		t.Fatal(err)
	}
	install(t, app, New().Defaults(app))
	if err := New().Uninstall(context.Background(), app, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(other); err != nil || string(data) != "someone else's" {
		t.Errorf("the other program's launcher changed: %q, %v", data, err)
	}
}

func TestXDGFoldersAreRespected(t *testing.T) {
	home := newHome(t)
	data := filepath.Join(home, "data")
	config := filepath.Join(home, "config")
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_CONFIG_HOME", config)
	if err := os.MkdirAll(filepath.Join(home, "Schreibtisch"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	dirs := "# written by xdg-user-dirs-update\nXDG_DESKTOP_DIR=\"$HOME/Schreibtisch\"\n"
	if err := os.WriteFile(filepath.Join(config, "user-dirs.dirs"), []byte(dirs), 0o644); err != nil {
		t.Fatal(err)
	}

	app := testApp()
	install(t, app, New().Defaults(app))
	for _, path := range []string{
		filepath.Join(data, testID, "knightloader"),
		filepath.Join(data, "applications", testID+".desktop"),
		filepath.Join(home, "Schreibtisch", testID+".desktop"),
	} {
		if !exists(path) {
			t.Errorf("%s is missing", path)
		}
	}
	if exists(filepath.Join(home, "Desktop", testID+".desktop")) {
		t.Error("the entry went to ~/Desktop rather than the configured desktop")
	}
}

func TestRelativeXDGDataHomeIsIgnored(t *testing.T) {
	home := newHome(t)
	t.Setenv("XDG_DATA_HOME", "data")
	if got, want := New().Defaults(testApp()).Dir, filepath.Join(home, ".local/share", testID); got != want {
		t.Errorf("Dir = %q, want %q", got, want)
	}
}

func TestXDGUserDirIsAskedWhenInstalled(t *testing.T) {
	home := newHome(t)
	desk := filepath.Join(home, "Arbeitsfläche")
	if err := os.Mkdir(desk, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(os.Getenv("PATH"), "xdg-user-dir")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\necho '"+desk+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := testApp()
	install(t, app, New().Defaults(app))
	if !exists(filepath.Join(desk, testID+".desktop")) {
		t.Error("no desktop copy where xdg-user-dir points")
	}
}

func TestNoDesktopCopyWithoutADesktopFolder(t *testing.T) {
	home := newHome(t)
	if err := os.Remove(filepath.Join(home, "Desktop")); err != nil {
		t.Fatal(err)
	}
	app := testApp()
	install(t, app, New().Defaults(app))
	if exists(filepath.Join(home, "Desktop")) {
		t.Error("the installer created a desktop folder")
	}
}

func TestCheckDirTakesOnlyTheFixedFolder(t *testing.T) {
	home := newHome(t)
	app := testApp()
	fixed := filepath.Join(home, ".local/share", testID)
	if chk := New().CheckDir(app, fixed+"/"); !chk.OK {
		t.Errorf("CheckDir(%s) = %+v, want OK", fixed, chk)
	}
	if chk := New().CheckDir(app, filepath.Join(home, "elsewhere")); chk.OK || chk.Code != setup.DirFixed {
		t.Errorf("CheckDir(elsewhere) = %+v, want %s", chk, setup.DirFixed)
	}
}

func TestLaunchStartsTheProgram(t *testing.T) {
	home := newHome(t)
	app := testApp()
	install(t, app, New().Defaults(app))
	if err := New().Launch(app); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "launched")
	for deadline := time.Now().Add(5 * time.Second); !exists(marker); {
		if time.Now().After(deadline) {
			t.Fatal("the program did not run")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestExecArgQuotesWhatTheEntryReserves(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"/home/a/.local/share/x/prog", "/home/a/.local/share/x/prog"},
		{"/home/a b/prog", `"/home/a b/prog"`},
		{"/home/$USER/prog", `"/home/\\$USER/prog"`},
		{`/home/a"b/prog`, `"/home/a\\"b/prog"`},
		{`/home/a\b/prog`, `"/home/a\\\\b/prog"`},
		{"/home/100%/prog", "/home/100%%/prog"},
	} {
		if got := execArg(tc.in); got != tc.want {
			t.Errorf("execArg(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// Package pommelporter installs a Wails desktop app on Windows and Linux.
// An app builds a small main package that fills in an App and calls Run;
// the result is one program that installs the app, shows its own window in
// the GlimStone look, and uninstalls it again.
package pommelporter

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// App describes the program an installer carries.
type App struct {
	// Name is shown in the window and names the folder, the shortcuts, the
	// entry under Apps and the update task.
	Name string
	// ID is the reverse-DNS identifier, such as com.junkerderprovinz.pommelexample.
	ID string
	// Executable is the program's file name inside Payload.
	Executable string
	Version    string
	Publisher  string
	Homepage   string
	// Accent is the app's GlimStone accent as a CSS colour.
	Accent string
	// Logo is the SVG the window shows.
	Logo []byte
	// Icon is a 256 px PNG for the Linux menu entry.
	Icon []byte
	// Categories place the Linux menu entry, such as Network and
	// FileTransfer; without any it goes under Utility.
	Categories []string
	// Payload holds the files to install, at the top of the install folder.
	Payload fs.FS
	// InstanceID is the Wails single-instance ID of a copy running on its
	// usual configuration, so the installer can ask it to quit. Empty when the
	// app runs no such lock.
	InstanceID string
	// Autostart is the value under HKCU\...\Run the app writes to start on
	// sign-in. One that named an earlier per-user copy is moved to the new one.
	Autostart string
	// Legacy names what earlier installers left behind.
	Legacy Legacy
	// Choices are ticks of the app's own on the options page, each unticked
	// until the person ticks it.
	Choices []Choice
	// Stays names what an uninstall leaves on the computer, by language code
	// such as "de", with "en" for a language it lacks. Without it the window
	// says the downloads, settings and accounts stay.
	Stays map[string]Stay
}

// Stay is what an uninstall leaves, in one language.
type Stay struct {
	// Item is the line under Kept on the uninstall page, such as "Your
	// settings and bookmarks".
	Item string
	// Note is the sentence after the uninstall, such as "Your settings and
	// bookmarks are still there."
	Note string
}

// Choice is a tick of the app's own on the options page, for something the
// installed program does itself, such as taking over a file type. After an
// install the program runs once with Do for each ticked choice, as the person
// signed in; before an uninstall it runs with Undo for every choice, ticked
// or not, so nothing it changed outside its folder outlives it. Do and Undo
// must be quick, show no window and do nothing when there is nothing to do.
type Choice struct {
	// ID names the choice in the options the window sends.
	ID string
	// Label is the text beside the tick by language code, such as "de";
	// "en" stands in for a language it lacks.
	Label map[string]string
	// Platforms limits the choice to these, "windows" or "linux"; empty
	// means everywhere.
	Platforms []string
	Do        []string
	Undo      []string
}

// Legacy names the registry keys of earlier installers, relative to
// HKLM or HKCU.
type Legacy struct {
	// MachineKeys are uninstall keys under HKLM; each one's uninstaller runs
	// silently before this one installs.
	MachineKeys []string
	// UserKeys are uninstall keys under HKCU of copies installed for one
	// person; their files, shortcuts and keys are removed.
	UserKeys []string
	// ChoiceKeys are where earlier installers kept the shortcut choice, read
	// when this installer has none of its own yet.
	ChoiceKeys []string
}

// Run installs or uninstalls app as the command line asks and returns when
// the window closes or the silent run ends.
//
//	/S or --silent      no window; installs with the defaults or last choice
//	/D=<folder>         the install folder for a silent install
//	--uninstall         removes the installed copy
func Run(app App) error {
	a := setup.App{
		Name: app.Name, ID: app.ID, Executable: app.Executable, Version: app.Version,
		Publisher: app.Publisher, Homepage: app.Homepage, Accent: app.Accent,
		Logo: app.Logo, Icon: app.Icon, Categories: app.Categories, Payload: app.Payload,
		InstanceID: app.InstanceID, Autostart: app.Autostart,
		Legacy: setup.Legacy(app.Legacy),
	}
	for _, c := range app.Choices {
		a.Choices = append(a.Choices, setup.Choice(c))
	}
	if len(app.Stays) > 0 {
		a.Stays = map[string]setup.Stay{}
		for lang, s := range app.Stays {
			a.Stays[lang] = setup.Stay(s)
		}
	}
	if a.Name == "" || a.Executable == "" || a.Payload == nil {
		return errors.New("pommelporter: App needs a Name, an Executable and a Payload")
	}
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}
	if args.uninstall {
		handled, code, err := handOver(os.Args[1:])
		if err != nil {
			return err
		}
		if handled {
			os.Exit(code)
		}
	}
	if err := prepare(a, args.silent, args.uninstall); err != nil {
		return err
	}
	p := platform()
	if args.silent {
		err = runSilent(p, a, args)
	} else {
		err = runWindow(p, a, args.uninstall)
	}
	if args.uninstall {
		code := 0
		if err != nil {
			code = 1
		}
		finishHandOver(code)
	}
	return err
}

type cmdline struct {
	silent    bool
	uninstall bool
	dir       string
}

// parseArgs reads NSIS's /S and /D= next to the usual flags, so deployment
// tools that know the old installer keep working.
func parseArgs(argv []string) (cmdline, error) {
	var c cmdline
	var rest []string
	for _, arg := range argv {
		switch {
		case arg == "/S":
			c.silent = true
		case len(arg) > 3 && arg[:3] == "/D=":
			c.dir = arg[3:]
		default:
			rest = append(rest, arg)
		}
	}
	fset := flag.NewFlagSet("installer", flag.ContinueOnError)
	fset.BoolVar(&c.silent, "silent", c.silent, "install or uninstall without a window")
	fset.BoolVar(&c.uninstall, "uninstall", false, "remove the installed copy")
	if err := fset.Parse(rest); err != nil {
		return c, err
	}
	return c, nil
}

func runSilent(p setup.Platform, a setup.App, c cmdline) error {
	ctx := context.Background()
	if c.uninstall {
		return p.Uninstall(ctx, a, setup.Removal{}, func(string) {})
	}
	o := p.Defaults(a)
	if c.dir != "" {
		o.Dir = c.dir
	}
	if chk := p.CheckDir(a, o.Dir); !chk.OK {
		return fmt.Errorf("%s cannot take %s: %s", o.Dir, a.Name, chk.Detail)
	}
	return p.Install(ctx, a, o, func(string) {})
}

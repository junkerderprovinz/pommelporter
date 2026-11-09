// Package setup holds what the platforms share: the app being installed, the
// choices made in the window, and the steps a platform reports while it works.
package setup

import (
	"context"
	"io/fs"
	"slices"
)

// App is the program an installer carries. See pommelporter.App, which the
// apps fill in and which is copied into this one.
type App struct {
	Name       string
	ID         string
	Executable string
	Version    string
	Publisher  string
	Homepage   string
	Accent     string
	Logo       []byte
	Icon       []byte
	Categories []string
	Payload    fs.FS
	InstanceID string
	Autostart  string
	Legacy     Legacy
	Choices    []Choice
	Stays      map[string]Stay
}

// Stay is what is left on the computer after an uninstall, in one language.
// See pommelporter.Stay.
type Stay struct {
	Item string `json:"item"`
	Note string `json:"note"`
}

// Choice is a tick of the app's own on the options page. See
// pommelporter.Choice.
type Choice struct {
	ID        string            `json:"id"`
	Label     map[string]string `json:"label"`
	Platforms []string          `json:"-"`
	Do        []string          `json:"-"`
	Undo      []string          `json:"-"`
}

// ChoicesOn are the app's choices that apply on platform, a name such as
// "windows".
func (a App) ChoicesOn(platform string) []Choice {
	var on []Choice
	for _, c := range a.Choices {
		if len(c.Platforms) == 0 || slices.Contains(c.Platforms, platform) {
			on = append(on, c)
		}
	}
	return on
}

// Ticked are the arguments to run the installed program with after an
// install, one list for each choice o ticked.
func (a App) Ticked(platform string, o Options) [][]string {
	var runs [][]string
	for _, c := range a.ChoicesOn(platform) {
		if o.Choices[c.ID] && len(c.Do) > 0 {
			runs = append(runs, c.Do)
		}
	}
	return runs
}

// Undos are the arguments to run the installed program with before an
// uninstall, for every choice whether or not it was ticked, since the person
// may have switched it on inside the app since.
func (a App) Undos(platform string) [][]string {
	var runs [][]string
	for _, c := range a.ChoicesOn(platform) {
		if len(c.Undo) > 0 {
			runs = append(runs, c.Undo)
		}
	}
	return runs
}

// Legacy names what earlier installers of an app left behind.
type Legacy struct {
	MachineKeys []string
	UserKeys    []string
	ChoiceKeys  []string
}

// Options are the choices on the options page.
type Options struct {
	Dir       string `json:"dir"`
	StartMenu bool   `json:"startMenu"`
	Desktop   bool   `json:"desktop"`
	// Language is the code the installed app starts in until someone picks
	// another one inside it, such as "de".
	Language string `json:"language"`
	// Choices holds the app's own choices by ID; one that is missing is not
	// ticked.
	Choices map[string]bool `json:"choices,omitempty"`
}

// Removal is the choice on the uninstall page. Without it an uninstall leaves
// everything in the person's own profile where it is.
type Removal struct {
	// WindowData also deletes what the app's window keeps for the person
	// uninstalling, such as the language and look picked in it. A test build
	// shares that folder with the real app, since both run the same program.
	WindowData bool `json:"windowData"`
}

// ValidLanguage reports whether code looks like a two-letter language code,
// so nothing else reaches the files an app reads it from.
func ValidLanguage(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// Existing describes a copy that is already installed.
type Existing struct {
	Version string `json:"version"`
	Dir     string `json:"dir"`
}

// DirCheck is the answer to whether a folder may take the program. Code names
// the reason for the window to translate, Detail says it for the log.
type DirCheck struct {
	OK     bool   `json:"ok"`
	Code   string `json:"code,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// The codes a DirCheck can carry.
const (
	DirRelative     = "relative"     // not an absolute path
	DirNotLocal     = "notLocal"     // a network, removable or optical drive
	DirNoACL        = "noAcl"        // a filesystem without access lists, such as FAT
	DirNotEmpty     = "notEmpty"     // holds files that are not an earlier copy of this app
	DirLink         = "link"         // the folder or one above it is a link or junction
	DirFile         = "file"         // a file has that name
	DirFixed        = "fixed"        // the platform installs into one folder only
)

// Folder is one directory as the folder chooser shows it.
type Folder struct {
	Path    string   `json:"path"`
	Parent  string   `json:"parent"`
	Roots   []string `json:"roots"`
	Entries []Entry  `json:"entries"`
}

// Entry is one sub-folder in a Folder.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// The steps a platform reports, in the order they run. The window shows them
// by these names, translated.
const (
	StepPrepare   = "prepare"
	StepClose     = "close"
	StepLegacy    = "legacy"
	StepFiles     = "files"
	StepShortcuts = "shortcuts"
	StepRegister  = "register"
	StepUpdates   = "updates"
	StepRemove    = "remove"
	StepDone      = "done"
)

// Report is called as each step starts.
type Report func(step string)

// Platform does the work on one operating system.
type Platform interface {
	// Defaults are the options a first install starts from, or the ones the
	// last install chose.
	Defaults(app App) Options
	// Existing is the installed copy, or nil.
	Existing(app App) *Existing
	// CheckDir says whether dir may take the program.
	CheckDir(app App, dir string) DirCheck
	// Roots are the tops of the filesystems the folder chooser offers.
	Roots() []string
	Install(ctx context.Context, app App, o Options, report Report) error
	Uninstall(ctx context.Context, app App, r Removal, report Report) error
	// Launch starts the installed program as the person who ran the
	// installer, not with the installer's rights.
	Launch(app App) error
}

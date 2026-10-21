// Package setup holds what the platforms share: the app being installed, the
// choices made in the window, and the steps a platform reports while it works.
package setup

import (
	"context"
	"io/fs"
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
	Payload    fs.FS
	InstanceID string
	Autostart  string
	Legacy     Legacy
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
	DirUnsafeParent = "unsafeParent" // a folder above it lets ordinary users rename or replace it
	DirLink         = "link"         // the folder or one above it is a link or junction
	DirFile         = "file"         // a file has that name
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
	Uninstall(ctx context.Context, app App, report Report) error
	// Launch starts the installed program as the person who ran the
	// installer, not with the installer's rights.
	Launch(app App) error
}

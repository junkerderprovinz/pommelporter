package pommelporter

import (
	"context"
	"sync"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
	"github.com/junkerderprovinz/pommelporter/ui"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// progressEvent carries each step to the window as {"step": name}, and a
// failure as {"error": text}.
const progressEvent = "progress"

// frameless leaves out the system's title bar on Windows, where the window
// draws its own. Linux keeps the frame its window manager draws.
const frameless = platformName == "windows"

// Installer is the service the window calls, as Call.ByName
// "github.com/junkerderprovinz/pommelporter.Installer.<Method>".
type Installer struct {
	p         setup.Platform
	a         setup.App
	uninstall bool
	app       *application.App
	window    *application.WebviewWindow

	mu   sync.Mutex
	busy bool
}

// Info is everything the window shows before anything happens.
type Info struct {
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Publisher string          `json:"publisher"`
	Homepage  string          `json:"homepage"`
	Accent    string          `json:"accent"`
	Logo      string          `json:"logo"`
	Mode      string          `json:"mode"`
	Platform  string          `json:"platform"`
	Existing  *setup.Existing `json:"existing"`
	Frameless bool            `json:"frameless"`
}

func (s *Installer) Info() Info {
	mode := "install"
	if s.uninstall {
		mode = "uninstall"
	}
	return Info{
		Name: s.a.Name, Version: s.a.Version, Publisher: s.a.Publisher, Homepage: s.a.Homepage,
		Accent: s.a.Accent, Logo: string(s.a.Logo), Mode: mode, Platform: platformName,
		Existing: s.p.Existing(s.a), Frameless: frameless,
	}
}

func (s *Installer) Defaults() setup.Options { return s.p.Defaults(s.a) }

func (s *Installer) CheckDir(dir string) setup.DirCheck { return s.p.CheckDir(s.a, dir) }

func (s *Installer) ListDir(path string) (setup.Folder, error) { return listDir(s.p, path) }

func (s *Installer) MakeDir(parent, name string) (string, error) { return makeDir(parent, name) }

// Install runs in the background and reports each step as a progress event;
// it returns at once.
func (s *Installer) Install(o setup.Options) {
	s.start(func(ctx context.Context, report setup.Report) error {
		if chk := s.p.CheckDir(s.a, o.Dir); !chk.OK {
			return &dirError{chk}
		}
		return s.p.Install(ctx, s.a, o, report)
	})
}

// Uninstall runs in the background like Install.
func (s *Installer) Uninstall() {
	s.start(func(ctx context.Context, report setup.Report) error {
		return s.p.Uninstall(ctx, s.a, report)
	})
}

func (s *Installer) Launch() error { return s.p.Launch(s.a) }

func (s *Installer) OpenURL(url string) error { return s.app.Browser.OpenURL(url) }

func (s *Installer) Minimise() { s.window.Minimise() }

func (s *Installer) Quit() { s.app.Quit() }

func (s *Installer) start(work func(context.Context, setup.Report) error) {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	s.busy = true
	s.mu.Unlock()
	go func() {
		defer func() { s.mu.Lock(); s.busy = false; s.mu.Unlock() }()
		report := func(step string) { s.app.Event.Emit(progressEvent, map[string]string{"step": step}) }
		if err := work(context.Background(), report); err != nil {
			s.app.Event.Emit(progressEvent, map[string]string{"error": err.Error()})
			return
		}
		report(setup.StepDone)
	}()
}

type dirError struct{ chk setup.DirCheck }

func (e *dirError) Error() string { return e.chk.Detail }

func runWindow(p setup.Platform, a setup.App, uninstall bool) error {
	svc := &Installer{p: p, a: a, uninstall: uninstall}
	title := a.Name
	app := application.New(application.Options{
		Name:     title,
		Services: []application.Service{application.NewService(svc)},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(ui.Dist)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	svc.app = app
	opts := application.WebviewWindowOptions{
		Title:               title,
		Width:               640,
		Height:              480,
		DisableResize:       true,
		MinimiseButtonState: application.ButtonEnabled,
		MaximiseButtonState: application.ButtonHidden,
	}
	if frameless {
		// The window is clear and the page draws its card with the logo on
		// the top edge, so it needs room above the card and for its shadow,
		// and none of the corners and shadow Windows would add.
		opts.Frameless = true
		opts.Height = 560
		opts.BackgroundType = application.BackgroundTypeTransparent
		opts.Windows.DisableFramelessWindowDecorations = true
	}
	svc.window = app.Window.NewWithOptions(opts)
	// Closing halfway would leave a half-installed copy behind, so the window
	// stays open until the work is done, whichever way it is closed.
	svc.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		if svc.busy {
			e.Cancel()
		}
	})
	return app.Run()
}

//go:build windows

package windows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

// legacyWait is how long an earlier uninstaller gets. NSIS uninstallers hand
// themselves to a copy and return at once, so the wait is for their entry to
// go.
const legacyWait = 60 * time.Second

// removeMachineCopies runs the uninstaller of each earlier installation for
// all users, silently, before anything of the new one is in place: its
// folder may be the new one or sit inside it.
func (p *platform) removeMachineCopies(ctx context.Context, app setup.App) error {
	for _, key := range app.Legacy.MachineKeys {
		cmdline := readString(p.machine, key, "UninstallString")
		if cmdline == "" {
			continue
		}
		cmd := exec.CommandContext(ctx, firstToken(cmdline))
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdline + " /S"}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("the uninstaller of the earlier %s failed: %w", app.Name, err)
		}
		deadline := time.Now().Add(legacyWait)
		for keyExists(p.machine, key) && time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return nil
}

// removeUserCopies removes a copy installed for the person running this,
// with its shortcuts and its entry under Apps. Only what an installation and
// its updates put there goes, since the folder could have been chosen by
// hand; the app's own settings stay. An autostart entry that named the old
// copy names exe instead.
func (p *platform) removeUserCopies(app setup.App, exe string) {
	for _, key := range app.Legacy.UserKeys {
		old := iconPath(readString(p.user, key, "DisplayIcon"))
		if old == "" {
			continue
		}
		p.moveAutostart(app, old, exe)

		dir := filepath.Dir(old)
		// The folder is the user's own. A link there could lead the
		// deletions below into a folder of the system.
		if attrs, err := attributes(dir); err == nil && !isLink(attrs) {
			removeSoon(old)
			removeSoon(old + ".old")
			os.Remove(filepath.Join(dir, "uninstall.exe"))
			removeMatches(filepath.Join(dir, ".*.update-*"))
			removeDirSoon(dir)
		}
		os.Remove(filepath.Join(p.places.userPrograms, app.Name+".lnk"))
		os.Remove(filepath.Join(p.places.userDesktop, app.Name+".lnk"))
		deleteKey(p.user, key)
		for _, choice := range app.Legacy.ChoiceKeys {
			deleteKey(p.user, choice)
		}
	}
}

func (p *platform) moveAutostart(app setup.App, old, exe string) {
	if app.Autostart == "" {
		return
	}
	k, err := registry.OpenKey(p.user, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	v, _, err := k.GetStringValue(app.Autostart)
	if err != nil || !strings.EqualFold(v, `"`+old+`"`) {
		return
	}
	k.SetStringValue(app.Autostart, `"`+exe+`"`)
}

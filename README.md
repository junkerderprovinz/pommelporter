# PommelPorter

The installer behind [KnightLoader](https://github.com/junkerderprovinz/knightloader), [ArrowLoop](https://github.com/junkerderprovinz/arrowloop) and [TrickWork](https://github.com/junkerderprovinz/trickwork). It installs a [Wails](https://wails.io) desktop app on Windows and Linux in a window of its own, in the GlimStone look, and uninstalls it again.

## What it does

**On Windows** it installs for everyone on the computer, under `C:\Program Files\<App>` or a folder you pick. It adds a Start menu entry and a desktop shortcut if you want them, an entry under Apps with its uninstaller, and a scheduled task that keeps the app up to date. A copy that is still running is asked to quit first, and installations left by the apps' earlier installers are taken over or removed.

The update task needs a folder nobody else can change. The installer gives the install folder the same rights as Program Files and checks every folder above it. On a drive where everyone may rename what sits at its top, as on many second drives, the task runs as the person who installed instead of as the system account, so swapping the folder gains nothing.

**On Linux** it installs for the current user under `~/.local/share/<app id>`, without a password, and adds a menu entry with the app's icon, a desktop icon if you want one, and an uninstall action in the menu entry.

**macOS** gets a disk image instead of an installer: `cmd/dmgbackground` draws the window background from the app's logo, and `scripts/make-dmg.sh` builds the image around it.

The window speaks the 42 languages the apps do, picked from the system's language, and hands that language to the app for its first start.

## Screenshots

<table>
  <tr>
    <td><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/welcome-dark.png"><img src=".github/assets/screenshots/welcome-light.png" alt="Start page with the app, its version and publisher"></picture></td>
    <td><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/options-dark.png"><img src=".github/assets/screenshots/options-light.png" alt="Install folder, Start menu entry and desktop shortcut"></picture></td>
  </tr>
  <tr>
    <td><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/progress-dark.png"><img src=".github/assets/screenshots/progress-light.png" alt="Installation in progress, step by step"></picture></td>
    <td><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/done-dark.png"><img src=".github/assets/screenshots/done-light.png" alt="Installation finished, with a button to start the app"></picture></td>
  </tr>
  <tr>
    <td><picture><source media="(prefers-color-scheme: dark)" srcset=".github/assets/screenshots/uninstall-dark.png"><img src=".github/assets/screenshots/uninstall-light.png" alt="Uninstall, with what goes and what stays"></picture></td>
    <td></td>
  </tr>
</table>

## Using it in an app

An app keeps a small main package next to its desktop module, embeds the built program as its payload, and calls `Run`:

```go
//go:embed payload
var payload embed.FS

func main() {
	files, _ := fs.Sub(payload, "payload")
	err := pommelporter.Run(pommelporter.App{
		Name:       "KnightLoader",
		ID:         "com.junkerderprovinz.knightloader",
		Executable: "KnightLoader.exe",
		Version:    version,
		Publisher:  "junkerderprovinz",
		Homepage:   "https://github.com/junkerderprovinz/knightloader",
		Accent:     "#FCC419",
		Logo:       logo,
		Icon:       icon,
		Payload:    files,
	})
	if err != nil {
		os.Exit(1)
	}
}
```

On Windows the installer needs a manifest that asks for administrator rights, linked in as a `.syso`. KnightLoader's `desktop/installer` and `scripts/desktop.mjs` show the whole build.

## Command line

| Switch | What it does |
| --- | --- |
| `/S` or `--silent` | Installs without a window, with the last choices or the defaults |
| `/D=<folder>` | The install folder for a silent install on Windows |
| `--uninstall` | Removes the installed copy; the entry under Apps calls the uninstaller this way |

## Building

The window lives in `ui/` (Vite, React, Tailwind) and is built into `ui/dist`, which is committed and embedded:

```sh
cd ui && npm ci && npm run build
go build ./...
go test ./...
```

`example/` is an installer for a stand-in app; CI uses it to install, check and uninstall for real on Windows and Linux.

## License

[AGPL-3.0](LICENSE)

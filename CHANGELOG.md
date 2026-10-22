# Changelog

Every release is written by hand. The full notes for each release are in
[`.github/release-notes/`](.github/release-notes/) and on the
[releases page](https://github.com/junkerderprovinz/pommelporter/releases).

## 1.0.4 - 2026-10-22

## 🎨 Design

- **The logo uses the family colours.** The pommel had its own gold and a plain grey, so it looked different from KnightLoader and ArrowLoop. It now uses the same gold and silver as they do, in the window icon, the README banner and the screenshots.

## 1.0.3 - 2026-10-22

## 🎨 Design

- **More room under the app tile.** The progress bar and the uninstall page started right below the tile. Both now sit in the middle of the window like the other pages, and a failed install still leaves space between the tile, the error and the buttons.

## 1.0.2 - 2026-10-21

## 🐛 Fixed

- **The folder picker no longer dims the desktop.** On Windows the window is clear around its card, and the dark scrim behind the folder picker covered that edge too, so a dark frame showed on the desktop around the installer. The scrim stays on the card.

## 1.0.1 - 2026-10-21

## 🎨 Design

- **The app's logo shows once.** The uninstall page and the last page put a second, smaller logo under the app tile at the top of the window. Only the tile is left.

## 1.0.0 - 2026-10-21

## ✨ Added

- **One installer for three apps.** An app fills in its name, logo and program file and gets an installer for Windows and Linux with a window in the GlimStone look.
- **Windows.** Installs for everyone under Program Files or a folder you pick, with Start menu entry, desktop shortcut, an entry under Apps and a daily update task. The installer gives the folder the rights Program Files has and checks every folder above it. Where ordinary users could swap the folder, the task runs as the person who installed instead of as SYSTEM.
- **Linux.** Installs for the current user under `~/.local/share`, with a menu entry and icon, an optional desktop icon and an uninstall action in the menu entry.
- **Updates and earlier installs.** A copy that is still running is asked to quit first. Installations left by the apps' earlier installers are taken over or removed, and an autostart entry moves to the new copy.
- **Silent use.** `/S` installs without a window, `/D=<folder>` picks the folder, and the uninstaller in the Apps entry accepts `/S` too.
- **macOS disk images.** `cmd/dmgbackground` draws the window background of a DMG from the app's logo, and `scripts/make-dmg.sh` builds the image around it.

## 🌐 Translations

- **The window speaks the 42 languages the apps do**, picked from the system's language.

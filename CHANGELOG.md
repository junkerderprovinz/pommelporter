# Changelog

Every release is written by hand. The full notes for each release are in
[`.github/release-notes/`](.github/release-notes/) and on the
[releases page](https://github.com/junkerderprovinz/pommelporter/releases).

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

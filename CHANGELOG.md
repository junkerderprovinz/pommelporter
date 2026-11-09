# Changelog

Every release is written by hand. The full notes for each release are in
[`.github/release-notes/`](.github/release-notes/) and on the
[releases page](https://github.com/junkerderprovinz/pommelporter/releases).

## 1.1.3 - 2026-11-09

## ✨ Added

- **An app can offer its own choices.** They show as unticked toggles on the options page. After the install the program runs once for each ticked choice, as the person signed in. Before an uninstall it runs again with each choice's undo arguments, so nothing it changed outside its folder stays behind.
- **An app can say what an uninstall leaves.** The uninstall page and the last page show the app's own line and sentence in each language. Without them the window keeps its words about downloads, settings and accounts.

## 🐛 Fixed

- **Uninstalling on Windows leaves nothing behind.** The uninstaller hands the work to a copy of itself in the temp folder and waits for it, so the install folder is gone right away. A new install into the same folder keeps its uninstaller, even before the next restart.
- **The window wears the app's icon on Linux.** GNOME shows it under the app's icon instead of a gear, during the install and up to the last page of an uninstall.

## 🌐 Translations

- **The Uninstall action in the app's menu entry** is named in the window's languages.

## 🎨 Design

- **The pommel has an open centre.** The new logo is in the window icon, the README banner and the screenshots.

## 🔧 Under the hood

- **A test checks the update folder:** users can change the settings in it and nothing else.

## 1.1.2 - 2026-11-06

## 🔧 Under the hood

- **The example installer is built with Go 1.27.3.**

## 1.1.1 - 2026-11-03

## 🎨 Design

- **The window follows GlimStone 3.9.0.** Cancel, Back, Browse and the other grey buttons take on a touch of the app's colour when the pointer or the keyboard reaches them, and their glyph turns that colour. The drive buttons in the folder picker do the same.

## 🔧 Under the hood

- **Wails 3.1.0** draws the window.

## 1.1.0 - 2026-10-31

## 🎨 Design

- **The window follows GlimStone 3.8.0.** Install, Try again, Create and Use this folder wear the same glyphs as in the apps. Minimise and Close look and behave like the buttons in the apps' title bar and show their name in a bubble. The progress bar only glides when your system allows motion.
- **The logo uses the family colours.** The pommel wears the gold and silver of KnightLoader and ArrowLoop, in the window icon, the README banner and the screenshots.
- **The app's logo shows once.** The uninstall page and the last page show the app tile and no second, smaller logo under it.
- **More room under the app tile.** The progress bar and the uninstall page sit in the middle of the window like the other pages, and a failed install leaves space between the tile, the error and the buttons.

## 🐛 Fixed

- **Uninstalling keeps what the app's window remembers.** On Windows the uninstaller deleted `%APPDATA%\<program>.exe`, where the app's window keeps the language and look you picked in it. A test build runs a program of the same name, so uninstalling just the test build wiped the real app's settings there too. The folder now stays. A switch on the uninstall page, off by default, deletes it as well.
- **The folder picker no longer dims the desktop.** On Windows its dark backdrop covered the clear edge around the installer and showed as a dark frame on the desktop. It stays on the card now.

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

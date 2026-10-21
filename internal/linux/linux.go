//go:build linux

// Package linux installs an app for the person running the installer, under
// ~/.local, with an entry in the desktop's application menu.
package linux

import "github.com/junkerderprovinz/pommelporter/internal/setup"

// New returns the Linux platform.
func New() setup.Platform { return nil }

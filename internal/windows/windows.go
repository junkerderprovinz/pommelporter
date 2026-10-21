//go:build windows

// Package windows installs an app for all users under Program Files or a
// folder only administrators can change.
package windows

import "github.com/junkerderprovinz/pommelporter/internal/setup"

// New returns the Windows platform.
func New() setup.Platform { return nil }

// Package ui holds the installer's window, built from ui/src into ui/dist.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist is the built window, rooted at its index.html.
var Dist, _ = fs.Sub(dist, "dist")

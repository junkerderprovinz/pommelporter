package pommelporter

import (
	"github.com/junkerderprovinz/pommelporter/internal/setup"
	"github.com/junkerderprovinz/pommelporter/internal/windows"
)

const platformName = "windows"

func platform() setup.Platform { return windows.New() }

func prepare(a setup.App, silent, uninstall bool) error {
	return windows.Prepare(a, silent, uninstall)
}

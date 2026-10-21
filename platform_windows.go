package pommelporter

import (
	"github.com/junkerderprovinz/pommelporter/internal/setup"
	"github.com/junkerderprovinz/pommelporter/internal/windows"
)

const platformName = "windows"

func platform() setup.Platform { return windows.New() }

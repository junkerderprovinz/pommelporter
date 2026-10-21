package pommelporter

import (
	"github.com/junkerderprovinz/pommelporter/internal/linux"
	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const platformName = "linux"

func platform() setup.Platform { return linux.New() }

func prepare(setup.App, bool, bool) error { return nil }

package pommelporter

import (
	"github.com/junkerderprovinz/pommelporter/internal/linux"
	"github.com/junkerderprovinz/pommelporter/internal/setup"
)

const platformName = "linux"

func platform() setup.Platform { return linux.New() }

func prepare(setup.App, bool, bool) error { return nil }

// handOver has nothing to do on Linux, where a running program's file can be
// deleted.
func handOver([]string) (bool, int, error) { return false, 0, nil }

func finishHandOver(int) {}

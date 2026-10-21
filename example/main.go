// Command example is an installer for a stand-in app, used to try the window
// and to run a whole install and uninstall in CI. It installs the files of the
// folder named by POMMELPORTER_PAYLOAD.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/junkerderprovinz/pommelporter"
)

//go:embed logo.svg
var logo []byte

func main() {
	err := pommelporter.Run(pommelporter.App{
		Name:       "PommelExample",
		ID:         "com.junkerderprovinz.pommelexample",
		Executable: executable,
		Version:    "1.0.0",
		Publisher:  "junkerderprovinz",
		Homepage:   "https://github.com/junkerderprovinz/pommelporter",
		Accent:     "#D9A44A",
		Logo:       logo,
		Payload:    os.DirFS(os.Getenv("POMMELPORTER_PAYLOAD")),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

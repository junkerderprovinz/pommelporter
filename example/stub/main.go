// Command stub stands in for the installed app in CI. Started with
// --fetch-extras it writes a few parts to the extras folder beside it and
// reports progress the way a component given by Run does, and otherwise it
// does nothing.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "--fetch-extras" {
		return
	}
	if err := os.MkdirAll("extras", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := 1; i <= 4; i++ {
		fmt.Printf("status Part %d of 4\n", i)
		time.Sleep(400 * time.Millisecond)
		if err := os.WriteFile(filepath.Join("extras", fmt.Sprintf("part-%d.txt", i)), nil, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("progress %.2f\n", float64(i)/4)
	}
}

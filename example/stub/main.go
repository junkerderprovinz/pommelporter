// Command stub stands in for the installed app in CI. Started with
// --fetch-extras it reports progress the way a component given by Run does,
// and otherwise does nothing.
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "--fetch-extras" {
		return
	}
	for i := 1; i <= 4; i++ {
		fmt.Printf("status Part %d of 4\n", i)
		time.Sleep(400 * time.Millisecond)
		fmt.Printf("progress %.2f\n", float64(i)/4)
	}
}

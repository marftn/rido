package cmd

import (
	"fmt"
	"os"
)

func RestoreCmd(files []string) {
	if len(files) < 1 {
		fmt.Println("Error: at least one file must be specified.")

		os.Exit(1)
	}
}

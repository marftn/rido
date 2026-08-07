package cmd

import (
	"fmt"
	"os"
)

func nilOrExit(errs ...error) {
	for _, err := range errs {
		if err != nil {
			fmt.Printf("Error: %v.\n", err)

			os.Exit(1)
		}
	}
}

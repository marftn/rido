package cmd

import (
	"os"
	"rido/internal/log"
)

func RestoreCmd(files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be specified.")

		os.Exit(1)
	}
}

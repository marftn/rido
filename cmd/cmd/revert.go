package cmd

import (
	"os"
	"rido/internal/config"
	"rido/internal/log"
)

func RevertCmd(_ config.Config, _ []string) {
	log.Error("The 'revert' command is not implemented yet.")

	os.Exit(1)
}

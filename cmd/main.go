package main

import (
	"os"
	"rido/cmd/cmd"
	"rido/internal/config"
	"rido/internal/log"
)

const (
	MinArgsNb = 2
)

func main() {
	if len(os.Args) < MinArgsNb {
		usage()

		os.Exit(1)
	}

	cfg := config.NewDummyConfig()

	switch os.Args[1] {
	case cmd.NameAddCmd:
		cmd.AddCmd(cfg, os.Args[2:])
	case cmd.NameListCmd:
		cmd.ListCmd(cfg, os.Args[2:])
	case cmd.NameRestoreCmd:
		cmd.RestoreCmd(cfg, os.Args[2:])
	case cmd.NameRevertCmd:
		cmd.RevertCmd(cfg, os.Args[2:])
	default:
		usage()

		os.Exit(1)
	}
}

func usage() {
	log.Info("Usage: tbd")
}

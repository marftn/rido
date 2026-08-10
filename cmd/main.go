package main

import (
	"os"
	"rido/cmd/cmd"
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

	switch os.Args[1] {
	case cmd.NameAddCmd:
		cmd.AddCmd(os.Args[2:])
	case cmd.NameListCmd:
		cmd.ListCmd(os.Args[2:])
	case cmd.NameRestoreCmd:
		cmd.RestoreCmd(os.Args[2:])
	case cmd.NameRevertCmd:
		cmd.RevertCmd(os.Args[2:])
	default:
		usage()

		os.Exit(1)
	}
}

func usage() {
	log.Info("Usage: tbd")
}

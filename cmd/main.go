package main

import (
	"fmt"
	"os"
	"rido/cmd/cmd"
)

func main() {
	if len(os.Args) < 2 {
		usage()

		os.Exit(1)
	}

	switch os.Args[1] {
	case cmd.NameCreateCmd:
		cmd.CreateCmd(os.Args[2:])
	case cmd.NameAddCmd:
		cmd.AddCmd(os.Args[2:])
	case cmd.NameListCmd:
		cmd.ListCmd(os.Args[2:])
	case cmd.NameRestoreCmd:
		cmd.RestoreCmd(os.Args[2:])
	case cmd.NameRmCmd:
		cmd.RmCmd(os.Args[2:])
	default:
		usage()

		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: tbd")
}

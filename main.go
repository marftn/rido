package main

import (
	"os"
	"runtime/debug"

	"github.com/marftn/rido/internal/cmd"
	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/log"
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
	case "version", "--version", "-v":
		log.Info(version())

		return
	case "help", "--help", "-h":
		usage()

		return
	}

	cfg, err := config.New()
	if err != nil {
		log.Errorf("Failed to load configuration: %v.", err)

		os.Exit(1)
	}

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

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "unknown"
	}

	return info.Main.Version
}

func usage() {
	log.Info(`Usage: rido <command> [flags] [path|id...]

  add <path>...             move files or directories into the store, leave symlinks
  list                      every store entry: ID, status, added, origin
  restore <path|id...>      recreate a symlink something removed
  revert <path|id...>       put the payload back and drop the entry
  version                   print the version of this binary
  help                      print this message

Flags for restore and revert:

  --all          every entry whose origin is under the current directory
  --store-wide   every entry in the store
  -f, --force    answer yes to every confirmation`)
}

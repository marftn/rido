package main

import (
	"errors"
	"os"
	"runtime/debug"

	"github.com/marftn/rido/internal/cmd"
	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/log"
)

const (
	MinArgsNb = 2

	HelpFlagShort = "-h"
	HelpFlagLong  = "--help"
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
	case "help", HelpFlagLong, HelpFlagShort:
		usage()

		return
	}

	if hasHelpFlag(os.Args[2:]) {
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
		err = cmd.AddCmd(cfg, os.Args[2:])
	case cmd.NameListCmd:
		err = cmd.ListCmd(cfg, os.Args[2:])
	case cmd.NameRestoreCmd:
		err = cmd.RestoreCmd(cfg, os.Args[2:])
	case cmd.NameRevertCmd:
		err = cmd.RevertCmd(cfg, os.Args[2:])
	default:
		usage()

		os.Exit(1)
	}

	if err == nil || errors.Is(err, cmd.ErrHelp) {
		return
	}

	if !errors.Is(err, cmd.ErrReported) {
		log.Errorf("%v.", err)
	}

	os.Exit(1)
}

// hasHelpFlag reports whether the user asked for help after the command name,
// so that `rido add -h` prints the same thing as `rido help`.
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == HelpFlagShort || arg == HelpFlagLong {
			return true
		}
	}

	return false
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

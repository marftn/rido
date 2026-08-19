package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"github.com/marftn/rido/internal/log"
	"github.com/marftn/rido/internal/store"
	"github.com/marftn/rido/internal/tty"
	"strings"
)

var (
	errNoTarget      = errors.New("at least one path or ID must be given, or --all")
	errBothTarget    = errors.New("--all and --store-wide take no path arguments")
	errEmptyStore    = errors.New("the store is empty")
	errFlagAfterPath = errors.New("flags must come before paths")
)

type commonFlags struct {
	all       bool
	storeWide bool
	force     bool
}

func parseFlags(name string, args []string) (commonFlags, []string) {
	var f commonFlags

	set := flag.NewFlagSet(name, flag.ExitOnError)
	set.BoolVar(&f.all, "all", false, "every entry whose origin is under the current directory")
	set.BoolVar(&f.storeWide, "store-wide", false, "every entry in the store")
	set.BoolVar(&f.force, "force", false, "answer yes to every confirmation")
	set.BoolVar(&f.force, "f", false, "shorthand for --force")

	// ExitOnError already exited if a flag was unknown or malformed.
	_ = set.Parse(args)

	return f, set.Args()
}

// targets is what a run acts on: the given paths and IDs, or the origins that
// --all (under the cwd) and --store-wide (the whole store) cover.
func targets(st *store.Store, f commonFlags, args []string) ([]string, error) {
	// flag.Parse stops at the first non-flag argument, so a flag after a path
	// would silently be taken for a path.
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("%w: %s", errFlagAfterPath, arg)
		}
	}

	if !f.all && !f.storeWide {
		if len(args) < 1 {
			return nil, errNoTarget
		}

		return args, nil
	}

	if len(args) > 0 {
		return nil, errBothTarget
	}

	scope := ""

	if !f.storeWide {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not get the current directory: %w", err)
		}

		scope = cwd
	}

	items := st.Under(scope)
	if len(items) == 0 {
		if scope == "" {
			return nil, errEmptyStore
		}

		return nil, fmt.Errorf(
			"no entries under %s (%d in the store, use --store-wide)",
			scope,
			len(st.Items),
		)
	}

	if outside := len(st.Items) - len(items); outside > 0 {
		log.Infof("%d store entries outside %s untouched", outside, scope)
	}

	origins := make([]string, 0, len(items))
	for _, item := range items {
		origins = append(origins, item.Meta.Origin)
	}

	return origins, nil
}

// confirm asks the user for confirmation, unless --force is set.
func confirm(force bool, format string, args ...any) (bool, error) {
	if force {
		return true, nil
	}

	if !tty.IsTTY() {
		return force, nil
	}

	isYes, err := tty.AskForConfirmation(os.Stdin, os.Stdout, format, args...)
	if err != nil {
		return false, fmt.Errorf("failed to get user confirmation: %w", err)
	}

	return isYes, nil
}

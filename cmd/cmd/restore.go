package cmd

import (
	"fmt"
	"os"
	"rido/internal/config"
	"rido/internal/fs"
	"rido/internal/log"
	"rido/internal/store"
	"rido/internal/tty"
)

func RestoreCmd(cfg config.Config, files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be specified.")

		os.Exit(1)
	}

	st, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	runEach(files, "restore", "restored", func(f string) error {
		return restoreOne(st, f)
	})
}

func restoreOne(st *store.Store, filename string) error {
	storeItem, err := st.FindStoreItem(filename)
	if err != nil {
		return err
	}

	switch status := storeItem.Status(); status {
	case store.StatusLinked:
		log.Infof("Already linked\t%s", storeItem.Meta.Origin)

		return nil
	case store.StatusMissing, store.StatusOccupied:
		return restoreFile(storeItem, status)
	case store.StatusStale:
		return fmt.Errorf(
			"origin directory of store item '%s' is missing: %s",
			storeItem.ID,
			storeItem.Meta.Origin,
		)
	case store.StatusBroken:
		return fmt.Errorf("payload is missing from store entry %s", storeItem.ID)
	default:
		return fmt.Errorf("unknown status %q", status)
	}
}

func restoreFile(storeItem *store.StoreItem, status store.Status) error {
	meta := storeItem.Meta
	was := "missing"

	if status == store.StatusOccupied {
		was = fs.Describe(meta.Origin)

		isYes, err := tty.AskForConfirmation(
			os.Stdin,
			os.Stdout,
			"'%s' is not our symlink (%s, modified %s). Delete it and relink?",
			meta.Origin,
			was,
			fs.ModifiedAgo(meta.Origin),
		)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if !isYes {
			return errSkipped
		}
	}

	err := store.ReplaceWithSymlink(meta, storeItem.PayloadPath())
	if err != nil {
		return err
	}

	log.Infof("Relinked\t%s\t(was %s)", meta.Origin, was)

	return nil
}

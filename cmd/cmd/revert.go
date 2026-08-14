package cmd

import (
	"errors"
	"fmt"
	"os"
	"rido/internal/config"
	"rido/internal/log"
	"rido/internal/store"
	"rido/internal/tty"
)

func RevertCmd(cfg config.Config, files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be specified.")

		os.Exit(1)
	}

	st, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	failed := 0

	for _, f := range files {
		e := revertOne(st, f)

		switch {
		case e == nil:
		case errors.Is(e, errSkipped):
			log.Infof("Skipped\t%s", f)

			failed++
		default:
			log.Errorf("Failed to revert '%s': %v.", f, e)

			failed++
		}
	}

	if failed > 0 {
		log.Errorf("%d could not be reverted.\t", failed)

		os.Exit(1)
	}
}

func revertOne(st *store.Store, filename string) error {
	storeItem, err := st.FindStoreItem(filename)
	if err != nil {
		return err
	}

	switch status := storeItem.Status(); status {
	case store.StatusLinked, store.StatusMissing, store.StatusOccupied, store.StatusStale:
		return revertFile(storeItem, status)
	case store.StatusBroken:
		return fmt.Errorf("payload is missing from store entry %s", storeItem.ID)
	default:
		return fmt.Errorf("unknown status %q", status)
	}
}

func revertFile(storeItem *store.StoreItem, status store.Status) error {
	meta := storeItem.Meta

	if status == store.StatusOccupied {
		isYes, err := tty.AskForConfirmation(
			os.Stdin,
			os.Stdout,
			"'%s' is not our symlink. Delete it and put the payload back?",
			meta.Origin,
		)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if !isYes {
			return errSkipped
		}
	}

	err := store.Revert(storeItem)
	if err != nil {
		return err
	}

	log.Infof("Reverted\t%s\t(entry %s dropped)", meta.Origin, storeItem.ID)

	return nil
}

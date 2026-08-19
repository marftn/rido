package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"
	"rido/internal/log"
	"rido/internal/store"
)

func RevertCmd(cfg config.Config, args []string) {
	flags, args := parseFlags(NameRevertCmd, args)

	st, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	files, err := targets(st, flags, args)
	if err != nil {
		log.Errorf("%v.", err)

		os.Exit(1)
	}

	runEach(files, "revert", "reverted", func(path string) error {
		return revertOne(st, path, flags.force)
	})
}

func revertOne(st *store.Store, filename string, force bool) error {
	storeItem, err := st.FindStoreItem(filename)
	if err != nil {
		return err
	}

	switch status := storeItem.Status(); status {
	case store.StatusLinked, store.StatusMissing, store.StatusOccupied, store.StatusStale:
		return revertFile(storeItem, status, force)
	case store.StatusBroken:
		return fmt.Errorf("payload is missing from store entry %s", storeItem.ID)
	default:
		return fmt.Errorf("unknown status %q", status)
	}
}

func revertFile(storeItem *store.StoreItem, status store.Status, force bool) error {
	meta := storeItem.Meta

	if status == store.StatusOccupied {
		isYes, err := confirm(
			force,
			"'%s' is not our symlink (%s, modified %s). Delete it and put the payload back?",
			meta.Origin,
			fs.Describe(meta.Origin),
			fs.ModifiedAgo(meta.Origin),
		)
		if err != nil {
			return err
		}

		if !isYes {
			return errSkipped
		}
	}

	if status == store.StatusStale {
		log.Infof("Recreating\t%s", filepath.Dir(meta.Origin))
	}

	err := store.Revert(storeItem)
	if err != nil {
		return err
	}

	log.Infof("Reverted\t%s\t(entry %s dropped)", meta.Origin, storeItem.ID)

	return nil
}

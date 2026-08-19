package cmd

import (
	"fmt"
	"os"
	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/log"
	"github.com/marftn/rido/internal/store"
)

func RestoreCmd(cfg config.Config, args []string) {
	flags, args := parseFlags(NameRestoreCmd, args)

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

	runEach(files, "restore", "restored", func(path string) error {
		return restoreOne(st, path, flags.force)
	})
}

func restoreOne(st *store.Store, filename string, force bool) error {
	storeItem, err := st.FindStoreItem(filename)
	if err != nil {
		return err
	}

	switch status := storeItem.Status(); status {
	case store.StatusLinked:
		log.Infof("Already linked\t%s", storeItem.Meta.Origin)

		return nil
	case store.StatusMissing, store.StatusOccupied:
		return restoreFile(storeItem, status, force)
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

func restoreFile(storeItem *store.StoreItem, status store.Status, force bool) error {
	meta := storeItem.Meta
	was := "missing"

	if status == store.StatusOccupied {
		was = fs.Describe(meta.Origin)

		isYes, err := confirm(
			force,
			"'%s' is not our symlink (%s, modified %s). Delete it and relink?",
			meta.Origin,
			was,
			fs.ModifiedAgo(meta.Origin),
		)
		if err != nil {
			return err
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

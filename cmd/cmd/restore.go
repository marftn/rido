package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/errors"
	"rido/internal/guard"
	"rido/internal/log"
	"rido/internal/store"
	"rido/internal/tty"
)

func RestoreCmd(cfg config.Config, files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be specified.")

		os.Exit(1)
	}

	store, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	for _, f := range files {
		storeItem, e := store.FindStoreItem(f)
		if errors.IsNotFound(e) {
			log.Debug("Could not find", f)

			continue
		} else if e != nil {
			log.Errorf("Could not find store item: %v.", e)

			continue
		}

		e = restoreFile(storeItem)
		if e != nil {
			log.Errorf("Failed to restore file '%s': %v.", f, e)

			continue
		}
	}
}

func restoreFile(storeItem *store.StoreItem) error {
	meta := storeItem.Meta
	if !guard.IsEmptyOrSymlink(meta.Origin) {
		isYes, err := tty.AskForConfirmation(
			os.Stdin,
			os.Stdout,
			"'%s' is a regular file or folder. Delete it and relink?",
			meta.Origin,
		)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if !isYes {
			log.Infof("Skipping file '%s'...", meta.Origin)

			return nil
		}
	}

	dstFile := filepath.Join(storeItem.Path(), meta.Filename)
	err := store.ReplaceWithSymlink(meta, dstFile)
	if err != nil {
		return err
	}

	log.Infof("Relinked\t%s", meta.Origin)

	return nil
}

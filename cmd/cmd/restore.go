package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
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

	store := store.NewStore(cfg)

	for _, f := range files {
		storeItem, err := store.FindStoreItem(f)
		if err != nil {
			log.Errorf("Could not find store item: %v.", err)

			continue
		} else if storeItem == nil {
			continue
		}

		restoreFile(storeItem)
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

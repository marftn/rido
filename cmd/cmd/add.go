package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/assert"
	"rido/internal/config"
	"rido/internal/log"
	"rido/internal/store"
)

func AddCmd(cfg config.Config, files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be added.")

		os.Exit(1)
	}

	nilOrExit(
		assert.AssertFilesExist(files),
		assert.AssertNoDuplicate(files),
		assert.AssertNoNestedPath(files),
	)

	store, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	for _, f := range files {
		e := addFile(store, f)
		if e != nil {
			log.Error(e)

			os.Exit(1)
		}
	}
}

func addFile(st *store.Store, filename string) error {
	origin, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("could not find origin: %w", err)
	}

	meta := store.Meta{
		Filename: filepath.Base(filename),
		Origin:   origin,
	}

	storeItem := st.NewStoreItem(&meta)

	log.Debug(meta)

	err = store.WriteStoreItem(&storeItem)
	if err != nil {
		return err
	}

	return nil
}

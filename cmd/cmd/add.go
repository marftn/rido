package cmd

import (
	"errors"
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
		assert.FilesExist(files),
		assert.NoDuplicate(files),
		assert.NoNestedPath(files),
	)

	st, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	failed := 0

	for _, f := range files {
		if e := addFile(st, f); e != nil {
			log.Errorf("Failed to add '%s': %v.", f, e)

			failed++

			continue
		}

		log.Infof("Added\t%s", f)
	}

	if failed > 0 {
		log.Errorf("%d could not be added.\n", failed)

		os.Exit(1)
	}
}

func addFile(st *store.Store, filename string) error {
	origin, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("could not find origin: %w", err)
	}

	_, err = st.FindStoreItem(origin)
	if err == nil {
		// Don't add a file if it's already managed, because it would orphan the previous store item.
		return errors.New("already managed by rido")
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	meta := store.Meta{
		Filename: filepath.Base(filename),
		Origin:   origin,
	}

	storeItem := st.NewStoreItem(&meta)

	log.Debug(meta)

	return store.WriteStoreItem(&storeItem)
}

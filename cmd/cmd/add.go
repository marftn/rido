package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/assert"
	"rido/internal/log"
	"rido/internal/store"
)

func AddCmd(files []string) {
	if len(files) < 1 {
		log.Error("At least one file must be added.")

		os.Exit(1)
	}

	nilOrExit(
		assert.AssertFilesExist(files),
		assert.AssertNoDuplicate(files),
		assert.AssertNoNestedPath(files),
	)

	for _, f := range files {
		err := addFile(f)
		if err != nil {
			log.Error(err)

			os.Exit(1)
		}
	}
}

func addFile(filename string) error {
	origin, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("could not find origin: %w", err)
	}

	meta := store.Meta{
		Filename: filepath.Base(filename),
		Origin:   origin,
	}

	storeItem := store.NewStoreItem(&meta)

	log.Debug(meta)

	err = store.WriteStoreItem(&storeItem)
	if err != nil {
		return err
	}

	return nil
}

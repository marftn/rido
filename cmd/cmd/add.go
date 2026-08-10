package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/assert"
	"rido/internal/store"
)

func AddCmd(files []string) {
	if len(files) < 1 {
		fmt.Println("Error: at least one file must be added.")

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
			fmt.Println("Error:", err)

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

	fmt.Println(meta)

	storeName := "/tmp/rido-store/" + storeItem.ID.String()
	err = os.MkdirAll(storeName, 0700)
	if err != nil {
		return fmt.Errorf("could not create store: %w", err)
	}

	err = store.WriteStoreItem(&storeItem)
	if err != nil {
		return err
	}

	return nil
}

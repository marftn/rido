package store

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"

	"github.com/oklog/ulid/v2"
)

type Store struct {
	Items []StoreItem
}

type StoreItem struct {
	ID   ulid.ULID
	Meta *Meta
}

func NewStoreItem(meta *Meta) StoreItem {
	return StoreItem{
		ID:   ulid.Make(),
		Meta: meta,
	}
}

func WriteStoreItem(storeItem *StoreItem) error {
	config := config.NewDummyConfig()

	storeItemFolder := filepath.Join(config.StoreLocation, storeItem.ID.String())
	err := os.MkdirAll(storeItemFolder, fs.DefaultPermissions)
	if err != nil {
		return fmt.Errorf("could not create store item folder: %w", err)
	}

	metaFilename := filepath.Join(storeItemFolder, MetaFilename)
	metaFile, err := os.Create(metaFilename)
	if err != nil {
		return fmt.Errorf("could not create meta file: %w", err)
	}

	defer metaFile.Close()

	_, err = WriteMetaFile(metaFile, storeItem.Meta)
	if err != nil {
		return err
	}

	return nil
}

package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"
)

const (
	MetaFilename string = "meta.json"
)

type Meta struct {
	Filename string `json:"filename"`
	Origin   string `json:"origin"`
}

func WriteMetaFile(file *os.File, meta Meta) (n int, err error) {
	data, err := json.Marshal(meta)
	if err != nil {
		err = fmt.Errorf("could not marshal meta to JSON: %w", err)

		return
	}

	n, err = file.Write(data)
	if err != nil {
		err = fmt.Errorf("could not write to meta file: %w", err)

		return
	}

	return
}

func CreateMetaFile(storeItem StoreItem) (*os.File, error) {
	config := config.NewDummyConfig()

	storeItemFolder := filepath.Join(config.StoreLocation, storeItem.ID.String())
	err := os.MkdirAll(storeItemFolder, fs.DefaultPermissions)
	if err != nil {
		return nil, fmt.Errorf("could not create store item folder: %w", err)
	}

	metaFilename := filepath.Join(storeItemFolder, MetaFilename)
	metaFile, err := os.Create(metaFilename)
	if err != nil {
		return nil, fmt.Errorf("could not create meta file: %w", err)
	}

	return metaFile, nil
}

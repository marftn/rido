package store

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"

	"github.com/oklog/ulid/v2"
)

const (
	tmpDirPattern = "rido-park-*"
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

	err = moveAndLink(*storeItem.Meta, storeItemFolder)
	if err != nil {
		return fmt.Errorf("could not move and link file: %w", err)
	}

	return nil
}

func moveAndLink(meta Meta, dstFolder string) error {
	// TODO:
	// - [x] Copy
	// - [ ] Verify
	// - [x] Park
	// - [x] Link
	// - [x] Remove
	dstFile := filepath.Join(dstFolder, meta.Filename)

	err := fs.CopyFile(dstFile, meta.Origin)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// TODO: Verify checksum.

	dir, err := os.MkdirTemp("", tmpDirPattern)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	parkedFile := filepath.Join(dir, meta.Filename)
	if e := os.Rename(meta.Origin, parkedFile); e != nil {
		return fmt.Errorf("failed to move %q to %q: %w", meta.Origin, parkedFile, e)
	}

	cleanup := func() error {
		if e := os.Rename(parkedFile, meta.Origin); e != nil {
			return fmt.Errorf("failed to restore parked file: %w", e)
		}

		return nil
	}

	err = os.Symlink(dstFile, meta.Origin)
	if err != nil {
		e := cleanup()
		if e != nil {
			err = fmt.Errorf("%w; cleanup failed: %w", err, e)
		}

		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// No need to manually remove `parkedFile`: it is done automatically by the deferred call to `os.RemoveAll`.
	return nil
}

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"
	"rido/internal/log"

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
	cleanup := func() {
		e := os.RemoveAll(storeItemFolder)
		if e != nil {
			log.Errorf("cleanup failed: %v", err)
		}
	}

	metaFilename := filepath.Join(storeItemFolder, MetaFilename)
	metaFile, err := os.Create(metaFilename)
	if err != nil {
		cleanup()

		return fmt.Errorf("could not create meta file: %w", err)
	}

	defer metaFile.Close()

	_, err = WriteMetaFile(metaFile, storeItem.Meta)
	if err != nil {
		cleanup()

		return err
	}

	err = moveAndLink(*storeItem.Meta, storeItemFolder)
	if err != nil {
		cleanup()

		return fmt.Errorf("could not move and link file: %w", err)
	}

	return nil
}

// moveAndLink moves the file described in the meta file to the store item
// and creates a symlink.
func moveAndLink(meta Meta, dstFolder string) error {
	dstFile := filepath.Join(dstFolder, meta.Filename)

	info, err := os.Lstat(meta.Origin)
	if err != nil {
		return fmt.Errorf("failed to lstat file: %w", err)
	}

	if info.IsDir() {
		err = fs.CopyDir(dstFile, meta.Origin)
		if err != nil {
			return fmt.Errorf("failed to copy dir: %w", err)
		}
	} else {
		err = fs.CopyFile(dstFile, meta.Origin)
		if err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// TODO: Verify checksum.

	parkDir, err := os.MkdirTemp("", tmpDirPattern)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(parkDir)

	parkFile := filepath.Join(parkDir, meta.Filename)
	if e := os.Rename(meta.Origin, parkFile); e != nil {
		return fmt.Errorf("failed to move %q to %q: %w", meta.Origin, parkFile, e)
	}

	cleanup := func() error {
		if e := os.Rename(parkFile, meta.Origin); e != nil {
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

	return nil
}

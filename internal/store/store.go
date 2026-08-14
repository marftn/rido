package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/config"
	"rido/internal/fs"
	"rido/internal/log"
	"time"

	"github.com/oklog/ulid/v2"
)

// parkSuffix is appended to the origin while it is moved aside. It is a sibling
// of the origin on purpose: the store is expected to live on another volume, so
// parking into a temp dir would fail with EXDEV.
const parkSuffix = ".rido-tmp"

var ErrNotFound = errors.New("store item not found")

// Status describes what sits at an entry's origin. `linked` is lowercase,
// anything needing a decision is uppercase.
type Status string

const (
	StatusLinked   Status = "linked"
	StatusMissing  Status = "MISSING"
	StatusOccupied Status = "OCCUPIED"
	StatusStale    Status = "STALE"
	StatusBroken   Status = "BROKEN"
)

type Store struct {
	Items  []StoreItem
	Config config.Config
}

type StoreItem struct {
	ID    ulid.ULID
	Meta  *Meta
	Store *Store
}

func NewStore(cfg config.Config) Store {
	return Store{
		Config: cfg,
		Items:  []StoreItem{},
	}
}

func LoadStore(cfg config.Config) (*Store, error) {
	st := NewStore(cfg)

	err := createStoreDir(cfg.StoreLocation)
	if err != nil {
		return nil, err
	}

	err = st.loadStoreItems()
	if err != nil {
		return nil, fmt.Errorf("could not load store items: %w", err)
	}

	return &st, nil
}

func (s *Store) NewStoreItem(meta *Meta) StoreItem {
	storeItem := StoreItem{
		ID:    ulid.Make(),
		Meta:  meta,
		Store: s,
	}

	s.Items = append(s.Items, storeItem)

	return storeItem
}

func (s *Store) FindStoreItem(filename string) (*StoreItem, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for '%s': %w", filename, err)
	}

	for _, storeItem := range s.Items {
		if storeItem.Meta.Origin == absPath {
			return &storeItem, nil
		}
	}

	return nil, ErrNotFound
}

func (s *StoreItem) Path() string {
	return filepath.Join(s.Store.Config.StoreLocation, s.ID.String())
}

// PayloadPath is the file or directory the origin symlink points at.
func (s *StoreItem) PayloadPath() string {
	return filepath.Join(s.Path(), s.Meta.Filename)
}

// Added is the entry's creation time, taken from its ULID.
func (s *StoreItem) Added() time.Time {
	return ulid.Time(s.ID.Time())
}

func (s *StoreItem) Status() Status {
	payload := s.PayloadPath()
	if !fs.Exists(payload) {
		return StatusBroken
	}

	info, err := os.Lstat(s.Meta.Origin)

	switch {
	case err == nil && info.Mode().Type() == os.ModeSymlink:
		if target, e := os.Readlink(s.Meta.Origin); e == nil && target == payload {
			return StatusLinked
		}

		return StatusOccupied
	case err == nil:
		return StatusOccupied
	case !os.IsNotExist(err):
		log.Errorf("failed to lstat '%s': %v", s.Meta.Origin, err)

		return StatusOccupied
	case !fs.Exists(filepath.Dir(s.Meta.Origin)):
		return StatusStale
	default:
		return StatusMissing
	}
}

func (s *StoreItem) String() string {
	return fmt.Sprintf(
		"%s\t%s\t%s\t%s",
		s.ID,
		s.Status(),
		s.Added().Format(time.DateOnly),
		s.Meta.Origin,
	)
}

func WriteStoreItem(storeItem *StoreItem) error {
	storeItemFolder := storeItem.Path()

	err := os.MkdirAll(storeItemFolder, fs.FileModeDefault)
	if err != nil {
		return fmt.Errorf("could not create store item folder: %w", err)
	}

	cleanup := func() {
		if e := os.RemoveAll(storeItemFolder); e != nil {
			log.Errorf("cleanup failed: %v", e)
		}
	}

	err = WriteMetaFile(filepath.Join(storeItemFolder, MetaFilename), storeItem.Meta)
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

// ReplaceWithSymlink points the origin `meta.Origina` at linkTarget. Whatever sits at
// the origin is moved aside first and only deleted once the symlink is in place, so a
// failure restores the starting state.
func ReplaceWithSymlink(meta *Meta, linkTarget string) error {
	parkFile := meta.Origin + parkSuffix
	parked := false

	if fs.Exists(meta.Origin) {
		if e := os.Rename(meta.Origin, parkFile); e != nil {
			return fmt.Errorf("failed to move %q to %q: %w", meta.Origin, parkFile, e)
		}

		parked = true
	}

	err := os.Symlink(linkTarget, meta.Origin)
	if err != nil {
		if parked {
			if e := os.Rename(parkFile, meta.Origin); e != nil {
				err = fmt.Errorf("%w; failed to restore parked file: %w", err, e)
			}
		}

		return fmt.Errorf("failed to create symlink: %w", err)
	}

	if parked {
		if e := os.RemoveAll(parkFile); e != nil {
			log.Warnf("could not remove parked file %q: %v", parkFile, e)
		}
	}

	return nil
}

func (s *Store) loadStoreItems() error {
	storePath := s.Config.StoreLocation

	entries, err := os.ReadDir(storePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			log.Warnf("Store contains non-folder item: '%s'. Skipping.", entry.Name())

			continue
		}

		id, e := ulid.Parse(entry.Name())
		if e != nil {
			log.Warnf("Store contains folder with non-ULID name: '%s'. Skipping.", entry.Name())

			continue
		}

		meta, e := LoadMetaFile(filepath.Join(storePath, entry.Name(), MetaFilename))
		if e != nil {
			log.Error(e)

			continue
		}

		storeItem := StoreItem{
			ID:    id,
			Meta:  meta,
			Store: s,
		}

		s.Items = append(s.Items, storeItem)
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

	err = ReplaceWithSymlink(&meta, dstFile)
	if err != nil {
		return err
	}

	return nil
}

func createStoreDir(storeLocation string) error {
	info, err := os.Lstat(storeLocation)

	switch {
	case os.IsNotExist(err):
		if e := os.MkdirAll(storeLocation, fs.FileModeDefault); e != nil {
			return fmt.Errorf("failed to create store folder: %w", e)
		}
	case err != nil:
		return fmt.Errorf("failed to lstat store folder '%s': %w", storeLocation, err)
	case !info.IsDir():
		return fmt.Errorf("'%s' exists and is not a folder", storeLocation)
	}

	return nil
}

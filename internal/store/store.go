package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/log"

	"github.com/oklog/ulid/v2"
)

// parkSuffix is appended to the origin while it is moved aside. It is a sibling
// of the origin on purpose: the store is expected to live on another volume, so
// parking into a temp dir would fail with EXDEV.
const parkSuffix = ".rido-tmp"

var (
	ErrNotFound  = errors.New("store item not found")
	ErrNotOrigin = errors.New("symlink is not the origin of its entry")

	ErrChecksumMismatch = errors.New("checksum mismatch after copy")
)

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

	err := createStoreDir(cfg.StoreRoot)
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

func (s *Store) FindStoreItem(target string) (*StoreItem, error) {
	if item, ok := s.getItemWithID(target); ok {
		return item, nil
	}

	item, err := s.getItemWithLink(target)
	if err != nil {
		return nil, err
	}

	if item != nil {
		return item, nil
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for '%s': %w", target, err)
	}

	for i := range s.Items {
		if s.Items[i].Meta.Origin == absPath {
			return &s.Items[i], nil
		}
	}

	return nil, ErrNotFound
}

func (s *Store) getItemWithID(target string) (*StoreItem, bool) {
	id, err := ulid.Parse(target)
	if err != nil {
		return nil, false
	}

	for i := range s.Items {
		if s.Items[i].ID == id {
			return &s.Items[i], true
		}
	}

	return nil, false
}

// getItemWithLink returns the item the symlink at `path` points into. A nil item
// and a nil error mean `path` is no symlink of ours, so resolution should carry on.
func (s *Store) getItemWithLink(path string) (*StoreItem, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return nil, nil //nolint:nilerr,nilnil // Not a symlink, so not our business.
	}

	linkedID := filepath.Base(filepath.Dir(target))

	item, ok := s.getItemWithID(linkedID)
	if !ok {
		return nil, nil //nolint:nilnil // Points elsewhere, so not our business.
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for '%s': %w", path, err)
	}

	if abs == item.Meta.Origin {
		return item, nil
	}

	// Acting on a file the user didn't attend to act on is wrong, so we return
	// an error to be sure and notify the user.
	if fs.Exists(item.Meta.Origin) {
		return nil, fmt.Errorf(
			"%w: '%s' is a second symlink to entry %s, whose origin is '%s'",
			ErrNotOrigin,
			abs,
			item.ID,
			item.Meta.Origin,
		)
	}

	// NOTE: The symlink we followed is the origin, whatever meta.json says.
	// A stale origin is only ever healed by hand.
	log.Warnf(
		"Origin '%s' of entry %s is gone, working on '%s' instead. Re-add the file to record it.",
		item.Meta.Origin,
		item.ID,
		abs,
	)

	item.Meta.Origin = abs

	return item, nil
}

// Under returns the entries whose origin is under dir. An empty dir means the
// whole store.
func (s *Store) Under(dir string) []StoreItem {
	if dir == "" {
		return s.Items
	}

	items := []StoreItem{}

	for _, storeItem := range s.Items {
		rel, err := filepath.Rel(dir, storeItem.Meta.Origin)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		items = append(items, storeItem)
	}

	return items
}

func (s *StoreItem) Path() string {
	return filepath.Join(s.Store.Config.StoreRoot, s.ID.String())
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
	status := s.Status()

	row := fmt.Sprintf(
		"%s\t%s\t%s\t%s",
		s.ID,
		status,
		s.Added().Format(time.DateOnly),
		s.Meta.Origin,
	)

	if detail := s.detail(status); detail != "" {
		row += "\t(" + detail + ")"
	}

	return row
}

// detail explains a status that needs a decision, e.g. "dir gone".
func (s *StoreItem) detail(status Status) string {
	switch status {
	case StatusOccupied:
		return fs.Describe(s.Meta.Origin)
	case StatusStale:
		return "dir gone"
	case StatusBroken:
		return "payload missing from store"
	case StatusLinked, StatusMissing:
		return ""
	default:
		return ""
	}
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

// ReplaceWithSymlink points the origin `meta.Origin` at linkTarget.
func ReplaceWithSymlink(meta *Meta, linkTarget string) error {
	return replaceOrigin(meta.Origin, func() error {
		if e := os.Symlink(linkTarget, meta.Origin); e != nil {
			return fmt.Errorf("failed to create symlink: %w", e)
		}

		return nil
	})
}

// Revert copies the payload back to its origin and drops the entry. The origin's
// directory is recreated if it is gone, so no entry can be stranded in the store.
func Revert(storeItem *StoreItem) error {
	origin := storeItem.Meta.Origin

	if e := os.MkdirAll(filepath.Dir(origin), fs.FileModeDefault); e != nil {
		return fmt.Errorf("could not recreate origin directory: %w", e)
	}

	err := replaceOrigin(origin, func() error {
		return copyPath(origin, storeItem.PayloadPath())
	})
	if err != nil {
		return err
	}

	if e := os.RemoveAll(storeItem.Path()); e != nil {
		return fmt.Errorf("could not drop store entry %s: %w", storeItem.ID, e)
	}

	return nil
}

// replaceOrigin moves whatever sits at the origin aside, runs write, and only deletes
// the parked copy once write succeeded, so a failure restores the starting state.
func replaceOrigin(origin string, write func() error) error {
	parkFile := origin + parkSuffix
	parked := false

	if fs.Exists(origin) {
		if e := os.Rename(origin, parkFile); e != nil {
			return fmt.Errorf("failed to move %q to %q: %w", origin, parkFile, e)
		}

		parked = true
	}

	err := write()
	if err != nil {
		if e := os.RemoveAll(origin); e != nil {
			err = fmt.Errorf("%w; failed to remove partial write: %w", err, e)
		}

		if parked {
			if e := os.Rename(parkFile, origin); e != nil {
				err = fmt.Errorf("%w; failed to restore parked file: %w", err, e)
			}
		}

		return err
	}

	if parked {
		if e := os.RemoveAll(parkFile); e != nil {
			log.Warnf("could not remove parked file %q: %v", parkFile, e)
		}
	}

	return nil
}

func (s *Store) loadStoreItems() error {
	storePath := s.Config.StoreRoot

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

	err := copyPath(dstFile, meta.Origin)
	if err != nil {
		return err
	}

	err = ReplaceWithSymlink(&meta, dstFile)
	if err != nil {
		return err
	}

	return nil
}

// copyPath copies a file or a directory tree, preserving modes and verifying that
// the copy is intact (checksum).
func copyPath(dst, src string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to lstat file: %w", err)
	}

	doCopy := fs.CopyFile
	if info.IsDir() {
		doCopy = fs.CopyDir
	}

	if e := doCopy(dst, src); e != nil {
		return fmt.Errorf("failed to copy %q: %w", src, e)
	}

	return verifyCopy(dst, src)
}

// verifyCopy compares the sha256 of the copy with the source.
func verifyCopy(dst, src string) error {
	srcSum, err := fs.Checksum(src)
	if err != nil {
		return err
	}

	dstSum, err := fs.Checksum(dst)
	if err != nil {
		return err
	}

	if srcSum != dstSum {
		return fmt.Errorf("%w: copied %q to %q", ErrChecksumMismatch, src, dst)
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

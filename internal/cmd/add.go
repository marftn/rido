package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/marftn/rido/internal/assert"
	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/fs"
	"github.com/marftn/rido/internal/git"
	"github.com/marftn/rido/internal/log"
	"github.com/marftn/rido/internal/store"
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

	runEach(files, "add", "added", func(f string) error {
		return addFile(st, f)
	})
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

	if e := git.Check(origin); git.IsErrNoTTY(e) {
		return errSkipped
	} else if e != nil {
		return e
	}

	meta := store.NewMeta(origin)

	storeItem := st.NewStoreItem(&meta)

	log.Debug(meta)

	err = store.WriteStoreItem(&storeItem)
	if err != nil {
		return err
	}

	line := fmt.Sprintf("Added\t%s\t%s", filename, storeItem.ID)
	if detail := fs.DescribeTree(storeItem.PayloadPath()); detail != "" {
		line += "\t(" + detail + ")"
	}

	log.Info(line)

	return nil
}

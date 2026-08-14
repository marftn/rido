package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"rido/internal/fs"
)

const (
	MetaFilename string = "meta.json"
)

const MetaVersion = 1

type Meta struct {
	Filename string `json:"filename"`
	Origin   string `json:"origin"`
	Version  int    `json:"v"`
}

func NewMeta(origin string) Meta {
	return Meta{
		Filename: filepath.Base(origin),
		Origin:   origin,
		Version:  MetaVersion,
	}
}

func WriteMetaFile(filename string, meta *Meta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("could not marshal meta to JSON: %w", err)
	}

	if e := os.WriteFile(filename, data, fs.FileModeReadOnly); e != nil {
		return fmt.Errorf("could not write meta file: %w", e)
	}

	return nil
}

func LoadMetaFile(filename string) (*Meta, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read meta file: %w", err)
	}

	var meta Meta

	err = json.Unmarshal(data, &meta)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal meta file '%s' (%q): %w", filename, data, err)
	}

	if meta.Version > MetaVersion {
		return nil, fmt.Errorf(
			"meta file '%s' is version %d, this version of rido only understands up to version %d",
			filename,
			meta.Version,
			MetaVersion,
		)
	}

	return &meta, nil
}

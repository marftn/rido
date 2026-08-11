package store

import (
	"encoding/json"
	"fmt"
	"os"
	"rido/internal/log"
)

const (
	MetaFilename string = "meta.json"
)

type Meta struct {
	Filename string `json:"filename"`
	Origin   string `json:"origin"`
}

func WriteMetaFile(file *os.File, meta *Meta) (n int, err error) {
	data, err := json.Marshal(*meta)
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

func LoadMetaFile(filename string) (*Meta, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read meta file: '%w'", err)
	}

	var meta Meta
	err = json.Unmarshal(data, &meta)
	if err != nil {
		log.Debug("Meta content:", data)
		return nil, fmt.Errorf("could not unmarshal meta file: %w", err)
	}

	return &meta, nil
}

package store

import (
	"encoding/json"
	"fmt"
	"os"
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

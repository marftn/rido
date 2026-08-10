package store

import (
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

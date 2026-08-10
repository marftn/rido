package store

import (
	"github.com/oklog/ulid/v2"
)

type Store struct {
	ID   ulid.ULID
	Meta *Meta
}

func NewStore(meta *Meta) Store {
	return Store{
		ID:   ulid.Make(),
		Meta: meta,
	}
}

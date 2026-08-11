package errors

import "errors"

var (
	ErrNotFound = errors.New("store item not found")
)

func IsNotFound(e error) bool {
	return errors.Is(e, ErrNotFound)
}

package git

import "errors"

var errNoTTY = errors.New("TTY not found")

func IsErrNoTTY(err error) bool {
	return errors.Is(err, errNoTTY)
}

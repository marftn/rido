package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNotFound(t *testing.T) {
	err := ErrNotFound
	otherErr := errors.New("test")

	require.True(t, IsNotFound(err))
	require.False(t, IsNotFound(otherErr))
}

package tty

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ask puts a canned answer to one confirmation and reports the verdict along
// with what the user would have seen.
func ask(t *testing.T, answer string) (bool, string) {
	t.Helper()

	var prompt strings.Builder

	isYes, err := AskForConfirmation(strings.NewReader(answer), &prompt, "delete '%s'?", ".env")
	require.NoError(t, err)

	return isYes, prompt.String()
}

// errReader stands in for a stdin that cannot be read at all, which is not the
// same as one that is simply empty.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestAskForConfirmationAcceptsYes(t *testing.T) {
	// "y" without a newline is the last line of a closed stdin.
	for _, answer := range []string{"y\n", "Y\n", "yes\n", "YES\n", "  y  \n", "y"} {
		isYes, _ := ask(t, answer)
		require.True(t, isYes, "answer %q should confirm", answer)
	}
}

func TestAskForConfirmationDeclinesEverythingElse(t *testing.T) {
	// "" is EOF on the first read: no TTY, or a user who closed the input.
	for _, answer := range []string{"n\n", "no\n", "ye\n", "maybe\n", "\n", ""} {
		isYes, _ := ask(t, answer)
		require.False(t, isYes, "answer %q should decline", answer)
	}
}

func TestAskForConfirmationMarksNoAsTheDefault(t *testing.T) {
	_, prompt := ask(t, "y\n")

	require.Equal(t, "delete '.env'? [y/N] ", prompt)
}

func TestAskForConfirmationFailsWhenTheAnswerCannotBeRead(t *testing.T) {
	closed := errors.New("stdin is gone")

	_, err := AskForConfirmation(errReader{err: closed}, io.Discard, "delete?")

	require.ErrorIs(t, err, closed)
}

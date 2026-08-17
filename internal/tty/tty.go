package tty

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func AskForConfirmation(in io.Reader, out io.Writer, format string, args ...any) (bool, error) {
	fmt.Fprintf(out, format+" [y/N] ", args...)

	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

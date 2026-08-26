package cmd

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/marftn/rido/internal/log"
)

const (
	NameAddCmd     = "add"
	NameListCmd    = "list"
	NameRestoreCmd = "restore"
	NameRevertCmd  = "revert"
)

var (
	errSkipped = errors.New("skipped")

	// ErrReported marks a failure whose details were already printed, so the
	// caller exits non-zero without adding another message.
	ErrReported = errors.New("reported")

	// ErrHelp means the command printed its own usage and nothing went wrong.
	ErrHelp = flag.ErrHelp
)

// runEach applies `do` to every path, printing one line per skipped or failed path
// and a final count. It returns ErrReported if any path was skipped or failed.
func runEach(files []string, verb, done string, do func(string) error) error {
	succeeded, skipped, failed := 0, 0, 0

	for _, f := range files {
		e := do(f)

		switch {
		case e == nil:
			succeeded++
		case errors.Is(e, errSkipped):
			log.Infof("Skipped\t%s", f)

			skipped++
		default:
			log.Errorf("Failed to %s '%s': %v.", verb, f, e)

			failed++
		}
	}

	counts := []string{fmt.Sprintf("%d %s", succeeded, done)}

	if skipped > 0 {
		counts = append(counts, fmt.Sprintf("%d skipped", skipped))
	}

	if failed > 0 {
		counts = append(counts, fmt.Sprintf("%d failed", failed))
	}

	log.Info(strings.Join(counts, ", "))

	if skipped+failed > 0 {
		return ErrReported
	}

	return nil
}

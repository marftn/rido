package cmd

import (
	"os"

	"github.com/marftn/rido/internal/log"
)

func nilOrExit(errs ...error) {
	for _, err := range errs {
		if err != nil {
			log.Errorf("Some errors occurred: %v.\n", err)

			os.Exit(1)
		}
	}
}

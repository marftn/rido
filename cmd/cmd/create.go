package cmd

import (
	"fmt"
	"os"
	"rido/internal/assert"
)

func CreateCmd(files []string) {
	if len(files) < 1 {
		fmt.Println("Error: at least one file must be specified.")

		os.Exit(1)
	}

	nilOrExit(
		assert.AssertFilesDoNotExist(files),
		assert.AssertNoDuplicate(files),
		assert.AssertNoNestedPath(files),
	)
}

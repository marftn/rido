package cmd

import (
	"fmt"
	"os"
	"rido/internal/assert"
)

func AddCmd(files []string) {
	if len(files) < 1 {
		fmt.Println("Error: at least one file must be added.")

		os.Exit(1)
	}

	nilOrExit(
		assert.AssertFilesExist(files),
		assert.AssertNoDuplicate(files),
		assert.AssertNoNestedPath(files),
	)
}

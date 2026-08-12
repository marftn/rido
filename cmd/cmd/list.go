package cmd

import (
	"fmt"
	"os"
	"rido/internal/config"
	"rido/internal/log"
	"rido/internal/store"
	"strings"
	"text/tabwriter"
)

func ListCmd(cfg config.Config, args []string) {
	if len(args) != 0 {
		log.Error("The 'list' command does not take any argument.")

		os.Exit(1)
	}

	store, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	err = displayTable(store.Items)
	if err != nil {
		log.Errorf("Failed to display list: %v", err)

		os.Exit(1)
	}
}

func displayTable(items []store.StoreItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	titleRow := strings.Join([]string{"ID", "STATUS", "ADDED", "ORIGIN"}, "\t")
	fmt.Fprintln(w, titleRow)

	for _, item := range items {
		fmt.Fprintln(w, item.String())
	}

	return w.Flush()
}

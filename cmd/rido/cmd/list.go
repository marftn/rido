package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/marftn/rido/internal/config"
	"github.com/marftn/rido/internal/log"
	"github.com/marftn/rido/internal/store"
)

const (
	padding = 2
)

func ListCmd(cfg config.Config, args []string) {
	if len(args) != 0 {
		log.Error("The 'list' command does not take any argument.")

		os.Exit(1)
	}

	st, err := store.LoadStore(cfg)
	if err != nil {
		log.Errorf("Failed to load store: %v.", err)

		os.Exit(1)
	}

	err = displayTable(st.Items)
	if err != nil {
		log.Errorf("Failed to display list: %v", err)

		os.Exit(1)
	}

	log.Infof("\n%s", summarize(st.Items))
}

// summarize counts entries by what they need.
// E.g. "6 entries: 2 linked, 3 need `rido restore`, 1 BROKEN".
func summarize(items []store.StoreItem) string {
	linked, needRestore, broken := 0, 0, 0

	for _, item := range items {
		switch item.Status() {
		case store.StatusLinked:
			linked++
		case store.StatusMissing, store.StatusOccupied, store.StatusStale:
			needRestore++
		case store.StatusBroken:
			broken++
		}
	}

	var counts []string

	if linked > 0 {
		counts = append(counts, fmt.Sprintf("%d linked", linked))
	}

	if needRestore > 0 {
		counts = append(counts, fmt.Sprintf("%d need `rido restore`", needRestore))
	}

	if broken > 0 {
		counts = append(counts, fmt.Sprintf("%d BROKEN", broken))
	}

	summary := fmt.Sprintf("%d entries", len(items))
	if len(counts) == 0 {
		return summary
	}

	return summary + ": " + strings.Join(counts, ", ")
}

func displayTable(items []store.StoreItem) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)
	titleRow := strings.Join([]string{"ID", "STATUS", "ADDED", "ORIGIN"}, "\t")
	fmt.Fprintln(w, titleRow)

	for _, item := range items {
		fmt.Fprintln(w, item.String())
	}

	return w.Flush()
}

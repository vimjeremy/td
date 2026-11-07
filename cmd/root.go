// Package cmd defines the td command-line interface.
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jeremy/td/internal/store"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "td",
	Short:         "A fast terminal todo list",
	Long:          "td keeps your tasks in a local SQLite file. Run it with no arguments for the interactive UI.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          func(*cobra.Command, []string) error { return runUI() },
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "td:", err)
		os.Exit(1)
	}
}

func openStore() (*store.Store, error) { return store.Open(store.DefaultPath()) }

func parseIDs(args []string) ([]int64, error) {
	ids := make([]int64, 0, len(args))
	for _, a := range args {
		id, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a task id", a)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

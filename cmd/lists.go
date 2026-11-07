package cmd

import (
	"fmt"

	"github.com/jeremy/td/internal/model"
	"github.com/jeremy/td/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(
		countCmd("projects", "List projects with open-task counts", "+", (*store.Store).Projects),
		countCmd("tags", "List tags with open-task counts", "@", (*store.Store).Tags),
		&cobra.Command{Use: "ui", Short: "Open the interactive UI", RunE: func(*cobra.Command, []string) error { return runUI() }},
	)
}

func countCmd(name, short, prefix string, list func(*store.Store) ([]model.Count, error)) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()
			items, err := list(st)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Printf("No %s yet.\n", name)
				return nil
			}
			for _, c := range items {
				if c.Total == 0 {
					continue // no task uses it any more
				}
				fmt.Printf("%-20s %d open\n", prefix+c.Name, c.Open)
			}
			return nil
		},
	}
}

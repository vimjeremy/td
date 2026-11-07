package cmd

import "github.com/jeremy/td/internal/ui"

func runUI() error {
	st, err := openStore()
	if err != nil {
		return err
	}
	defer st.Close()
	return ui.Run(st)
}

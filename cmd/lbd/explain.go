package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Re-display the most recent .lbd-report.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("lbd explain: not implemented yet")
			return nil
		},
	}
}
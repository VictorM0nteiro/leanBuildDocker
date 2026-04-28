package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Print the structured project analysis as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("lbd analyze: not implemented yet")
			return nil
		},
	}
}
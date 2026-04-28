package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use: "doctor",
		Short: "Diagnose potential issues with a project or its image",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("lbd doctor: not implemented yet(Phase 7)")
			return nil
		},
	}
}
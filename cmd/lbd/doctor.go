package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/VictorM0nteiro/leanBuildDocker/internal/doctor"
)

func newDoctorCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose potential issues with a project and its Dockerfile",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := dir
			var err error
			if target == "" {
				target, err = resolveProjectRoot()
			} else {
				target, err = filepath.Abs(target)
			}
			if err != nil {
				return err
			}

			rep, err := doctor.Diagnose(target)
			if err != nil {
				return err
			}
			printDoctorReport(os.Stdout, rep)

			for _, f := range rep.Findings {
				if f.Severity == doctor.SeverityError {
					return fmt.Errorf("doctor found issues that need attention")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "target", "t", "", "directory to diagnose (default: current directory)")
	return cmd
}

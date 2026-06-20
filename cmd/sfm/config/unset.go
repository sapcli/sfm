package config

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

func init() {
	Cmd.AddCommand(unsetCmd)
}

var unsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Remove saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sapme.ClearConfig(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Saved credentials removed.")
		return nil
	},
}

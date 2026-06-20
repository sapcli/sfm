package config

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

func init() {
	Cmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Show saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := sapme.ReadConfig()
		if err != nil {
			return err
		}
		if cfg == nil || cfg.Username == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "No credentials configured.")
			return nil
		}
		masked := ""
		if cfg.Password != "" {
			masked = "********"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "username: %s\npassword: %s\n", cfg.Username, masked)
		return nil
	},
}

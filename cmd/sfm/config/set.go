package config

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

var setUsername string
var setPassword string

func init() {
	Cmd.AddCommand(setCmd)
	setCmd.Flags().StringVar(&setUsername, "username", "", "SAP S-User ID")
	setCmd.Flags().StringVar(&setPassword, "password", "", "SAP password")
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set saved credentials",
	Long: `Persist username and password to the config file.

One of --username or --password must be provided. Only the provided
fields are updated — omitted fields keep their current value.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if setUsername == "" && setPassword == "" {
			return fmt.Errorf("at least one of --username or --password is required")
		}
		cfg, err := sapme.ReadConfig()
		if err != nil {
			return err
		}
		if cfg == nil {
			cfg = &sapme.Config{}
		}
		if setUsername != "" {
			cfg.Username = setUsername
		}
		if setPassword != "" {
			cfg.Password = setPassword
		}
		if err := sapme.WriteConfig(cfg); err != nil {
			return err
		}
		path, _ := sapme.ConfigFilePath()
		fmt.Fprintf(cmd.OutOrStdout(), "Credentials saved to %s\n", path)
		return nil
	},
}

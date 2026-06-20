package config

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration (credentials)",
	Long: `Manage persistent CLI configuration.

Store SAP credentials in a config file so they don't need
to be passed via --username/--password flags or environment
variables every time.

Precedence (highest to lowest):
  1. --username / --password flags
  2. SAP_ADMIN_USERNAME / SAP_ADMIN_PASSWORD env vars
  3. Config file (saved via 'sfm config set')
`,
}

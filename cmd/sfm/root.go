package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
	"github.com/sapcli/sfm/cmd/sfm/config"
	"github.com/sapcli/sfm/cmd/sfm/partneruser"
	"github.com/sapcli/sfm/cmd/sfm/user"
)

var (
	username     string
	password     string
	timeout      time.Duration
	httpLogLevel string
	debugBodyMax int
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:           "sfm",
	Short:         "SAP For Me CLI - Manage users and partner users",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := sapme.ValidateOutputFormat(outputFormat); err != nil {
			return err
		}
		if needsAuth(cmd) {
			if username == "" {
				username = os.Getenv("SAP_ADMIN_USERNAME")
			}
			if password == "" {
				password = os.Getenv("SAP_ADMIN_PASSWORD")
			}
			if httpLogLevel == "" {
				httpLogLevel = os.Getenv("HTTP_LOG_LEVEL")
			}
			// Fall back to config file if flags and env are empty.
			if username == "" || password == "" {
				if cfg, err := sapme.ReadConfig(); err == nil && cfg != nil {
					if username == "" {
						username = cfg.Username
					}
					if password == "" {
						password = cfg.Password
					}
				}
			}
			if username == "" || password == "" {
				return fmt.Errorf("username/password required (flags, env, or config)")
			}
		}
		return nil
	},
}

func needsAuth(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "help" || c.Name() == "completion" || c.Name() == "config" {
			return false
		}
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "SAP SID (or SAP_ADMIN_USERNAME env)")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "SAP password (or SAP_ADMIN_PASSWORD env)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 90*time.Second, "request timeout")
	rootCmd.PersistentFlags().StringVar(&httpLogLevel, "http-log-level", "", "HTTP log level: debug|info|warn|error")
	rootCmd.PersistentFlags().IntVar(&debugBodyMax, "debug-body-max", 2048, "max body bytes to log")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "output format: json|text|table")

	sapme.Username = &username
	sapme.Password = &password
	sapme.Timeout = &timeout
	sapme.HTTPLogLevel = &httpLogLevel
	sapme.DebugBodyMax = &debugBodyMax
	sapme.OutputFormat = &outputFormat

	_ = rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return sapme.ValidOutputFormats, cobra.ShellCompDirectiveDefault
	})

	rootCmd.AddCommand(
		config.Cmd,
		partneruser.Cmd,
		user.Cmd,
	)
}

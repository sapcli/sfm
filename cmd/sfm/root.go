package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
	"github.com/sapcli/sfm/cmd/sfm/user"
	"github.com/sapcli/sfm/cmd/sfm/partneruser"
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
		if username == "" {
			username = os.Getenv("SAP_ADMIN_USERNAME")
		}
		if password == "" {
			password = os.Getenv("SAP_ADMIN_PASSWORD")
		}
		if httpLogLevel == "" {
			httpLogLevel = os.Getenv("HTTP_LOG_LEVEL")
		}
		if username == "" || password == "" {
			return fmt.Errorf("username/password required (flags or env SAP_ADMIN_USERNAME/SAP_ADMIN_PASSWORD)")
		}
		return nil
	},
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

	rootCmd.AddCommand(
		user.Cmd,
		partneruser.Cmd,
	)
}

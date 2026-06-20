// Package user implements the "sfm user" command tree for managing SAP For Me users.
// Subcommands: list, get, create, delete, extend, search.
package user

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

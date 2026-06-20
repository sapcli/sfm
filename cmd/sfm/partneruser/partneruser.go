// Package partneruser implements the "sfm partneruser" command tree for managing
// SAP partner users. Subcommands: list, create, delete, search.
package partneruser

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "partneruser",
	Short: "Manage partner users",
}

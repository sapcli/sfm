package user

import (
	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

func init() {
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := sapme.MustClient()
		results, err := client.UserAdmin().Users(cmd.Context())
		if err != nil {
			return err
		}
		sapme.Print(results)
		return nil
	},
}

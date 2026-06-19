package user

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

var getUserID string

func init() {
	Cmd.AddCommand(getCmd)
	getCmd.Flags().StringVar(&getUserID, "user-id", "", "user ID (required)")
	_ = getCmd.MarkFlagRequired("user-id")
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a user by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		if getUserID == "" {
			return fmt.Errorf("--user-id is required")
		}
		client := sapme.MustClient()
		user, err := client.UserAdmin().GetUser(cmd.Context(), getUserID)
		if err != nil {
			return err
		}
		sapme.Print(user)
		return nil
	},
}

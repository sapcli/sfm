package user

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

var deleteUserID string

func init() {
	Cmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVar(&deleteUserID, "user-id", "", "user ID (required)")
	_ = deleteCmd.MarkFlagRequired("user-id")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteUserID == "" {
			return fmt.Errorf("--user-id is required")
		}
		client := sapme.MustClient()
		if err := client.UserAdmin().Delete(cmd.Context(), deleteUserID); err != nil {
			return err
		}
		sapme.Print(map[string]any{"ok": true, "message": "user deleted"})
		return nil
	},
}

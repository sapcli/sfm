package partneruser

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

var deleteEmail string

func init() {
	Cmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringVar(&deleteEmail, "email", "", "email of partner user to delete (required)")
	_ = deleteCmd.MarkFlagRequired("email")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a partner user by email",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteEmail == "" {
			return fmt.Errorf("--email is required")
		}
		client := sapme.MustClient()
		if err := client.Partner().Auth(cmd.Context()); err != nil {
			return err
		}
		_, err := client.Partner().DeleteByEmail(cmd.Context(), deleteEmail)
		return err
	},
}

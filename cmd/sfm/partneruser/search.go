package partneruser

import (
	"fmt"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

var searchEmail string

func init() {
	Cmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVar(&searchEmail, "email", "", "email to search (required)")
	_ = searchCmd.MarkFlagRequired("email")
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search partner users by email",
	RunE: func(cmd *cobra.Command, args []string) error {
		if searchEmail == "" {
			return fmt.Errorf("--email is required")
		}
		client := sapme.MustClient()
		if err := client.Partner().Auth(cmd.Context()); err != nil {
			return err
		}
		results, err := client.Partner().Search(cmd.Context(), searchEmail)
		if err != nil {
			return err
		}
		sapme.Print(results)
		return nil
	},
}

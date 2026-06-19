package user

import (
	"fmt"

	"github.com/spf13/cobra"
	sfm "github.com/sapcli/sfm"
	sapme "github.com/sapcli/sfm/cmd/sfm/internal"
)

var (
	searchKeyword string
	searchField   string
	searchCustomerID string
)

func init() {
	Cmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVar(&searchKeyword, "keyword", "", "search keyword (required)")
	searchCmd.Flags().StringVar(&searchField, "field", "Ipadr", "search field")
	searchCmd.Flags().StringVar(&searchCustomerID, "customer-id", "", "customer ID")
	_ = searchCmd.MarkFlagRequired("keyword")
}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search users by keyword",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := sapme.MustClient()
		results, err := client.UserAdmin().Search(cmd.Context(), searchKeyword, sfm.SearchOption{
			Field:      searchField,
			CustomerID: searchCustomerID,
		})
		if err != nil {
			return err
		}
		sapme.Print(results)
		return nil
	},
}

func searchCmdE(cmd *cobra.Command, args []string) error {
	if searchKeyword == "" {
		return fmt.Errorf("--keyword is required")
	}
	return nil
}

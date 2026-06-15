package partneruser

import (
	"fmt"

	"github.com/spf13/cobra"
	launchpad "github.com/sapcli/me"
	sapme "github.com/sapcli/me/cmd/sapme/internal"
)

var (
	createEmail     string
	createFirstName string
	createLastName  string
	createExpireMS  int64
	createContactID string
	createDuplicate bool
)

func init() {
	Cmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createEmail, "email", "", "email address (required)")
	createCmd.Flags().StringVar(&createFirstName, "first-name", "", "first name")
	createCmd.Flags().StringVar(&createLastName, "last-name", "", "last name")
	createCmd.Flags().Int64Var(&createExpireMS, "expire-ms", 0, "expiry timestamp in milliseconds")
	createCmd.Flags().StringVar(&createContactID, "contact-id", "", "contact ID")
	createCmd.Flags().BoolVar(&createDuplicate, "check-duplicate", true, "check for duplicates")
	_ = createCmd.MarkFlagRequired("email")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new partner user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if createEmail == "" {
			return fmt.Errorf("--email is required")
		}
		client := sapme.MustClient()
		if err := client.Partner().Auth(cmd.Context()); err != nil {
			return err
		}
		user, err := client.Partner().Create(cmd.Context(), launchpad.CreatePartnerUserRequest{
			Email:          createEmail,
			FirstName:      createFirstName,
			LastName:       createLastName,
			ExpireDateMS:   createExpireMS,
			ContactID:      createContactID,
			CheckDuplicate: createDuplicate,
		})
		if err != nil {
			return err
		}
		sapme.Print(user)
		return nil
	},
}

package user

import (
	"fmt"

	"github.com/spf13/cobra"
	sfm "github.com/sapcli/sfm"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

var (
	createEmail      string
	createFirstName  string
	createLastName   string
	createSalutation string
	createCustomerID string
	createDeptID     string
	createLanguage   string
)

func init() {
	Cmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createEmail, "email", "", "email address (required)")
	createCmd.Flags().StringVar(&createFirstName, "first-name", "", "first name")
	createCmd.Flags().StringVar(&createLastName, "last-name", "", "last name")
	createCmd.Flags().StringVar(&createSalutation, "salutation", "Mr", "salutation (Mr/Mrs/Ms)")
	createCmd.Flags().StringVar(&createCustomerID, "customer-id", "", "customer ID")
	createCmd.Flags().StringVar(&createDeptID, "department-id", "", "department ID")
	createCmd.Flags().StringVar(&createLanguage, "language", "EN", "language code")
	_ = createCmd.MarkFlagRequired("email")
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if createEmail == "" {
			return fmt.Errorf("--email is required")
		}
		client := sapme.MustClient()
		user, err := client.UserAdmin().Create(cmd.Context(), sfm.CreateUserRequest{
			Email:        createEmail,
			FirstName:    createFirstName,
			LastName:     createLastName,
			Salutation:   createSalutation,
			CustomerID:   createCustomerID,
			DepartmentID: createDeptID,
			Language:     createLanguage,
		})
		if err != nil {
			return err
		}
		sapme.Print(user)
		return nil
	},
}

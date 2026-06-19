package user

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

var (
	extendUserIDs string
	extendDays    int
)

func init() {
	Cmd.AddCommand(extendCmd)
	extendCmd.Flags().StringVar(&extendUserIDs, "user-ids", "", "comma-separated user IDs (required)")
	extendCmd.Flags().IntVar(&extendDays, "days", 90, "number of days to extend")
	_ = extendCmd.MarkFlagRequired("user-ids")
}

var extendCmd = &cobra.Command{
	Use:   "extend",
	Short: "Extend user expiry date",
	RunE: func(cmd *cobra.Command, args []string) error {
		if extendUserIDs == "" {
			return fmt.Errorf("--user-ids is required")
		}
		userIDs := strings.Split(extendUserIDs, ",")
		for i := range userIDs {
			userIDs[i] = strings.TrimSpace(userIDs[i])
		}
		client := sapme.MustClient()
		results, err := client.UserAdmin().ExtendExpiryDate(cmd.Context(), userIDs, extendDays)
		if err != nil {
			return err
		}
		sapme.Print(results)
		return nil
	},
}

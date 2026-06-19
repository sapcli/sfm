package user

import (
	"github.com/spf13/cobra"
	sfm "github.com/sapcli/sfm"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

var (
	requestsStatus    string
	requestsType      string
	requestsRequester string
	requestsProcessor string
)

func init() {
	Cmd.AddCommand(requestsCmd)
	requestsCmd.Flags().StringVar(&requestsStatus, "status", "", "filter by status (O=Open, R=Rejected, A=Approved)")
	requestsCmd.Flags().StringVar(&requestsType, "type", "", "filter by request type")
	requestsCmd.Flags().StringVar(&requestsRequester, "requester", "", "filter by requester")
	requestsCmd.Flags().StringVar(&requestsProcessor, "processor", "", "filter by processor")
}

var requestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List user requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := sapme.MustClient()
		results, err := client.UserAdmin().GetUserRequests(cmd.Context(), sfm.RequestFilter{
			Status:    requestsStatus,
			Type:      requestsType,
			Requester: requestsRequester,
			Processor: requestsProcessor,
		})
		if err != nil {
			return err
		}
		sapme.Print(map[string]any{"count": len(results), "results": results})
		return nil
	},
}

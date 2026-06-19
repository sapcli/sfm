package partneruser

import (
	"github.com/spf13/cobra"
	sapme "github.com/sapcli/sfm/cmd/sapme/internal"
)

func init() {
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all partner users",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := sapme.MustClient()
		if err := client.Partner().Auth(cmd.Context()); err != nil {
			return err
		}
		results, err := client.Partner().Users(cmd.Context())
		if err != nil {
			return err
		}
		sapme.Print(map[string]any{"count": len(results), "results": results})
		return nil
	},
}

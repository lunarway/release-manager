package policy

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewDelete(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "delete",
		Short: "Delete one or more policies",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}

			var resp httpinternal.DeletePolicyResponse
			path, err := client.URL(path)
			if err != nil {
				return err
			}
			err = client.Req(http.MethodDelete, path, httpinternal.DeletePolicyRequest{
				Service:        *service,
				PolicyIDs:      args,
				CommitterName:  committerName,
				CommitterEmail: committerEmail,
			}, &resp)
			if err != nil {
				return err
			}
			fmt.Printf("Deleted %d policies\n", resp.Count)
			return nil
		},
	}
	return command
}

package command

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewPromote(client *httpinternal.Client, service *string) *cobra.Command {
	var environment, namespace string
	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}

			var resp httpinternal.PromoteResponse
			url, err := client.URL("promote")
			if err != nil {
				return err
			}
			err = client.Do(http.MethodPost, url, httpinternal.PromoteRequest{
				Service:        *service,
				Environment:    environment,
				Namespace:      namespace,
				CommitterName:  committerName,
				CommitterEmail: committerEmail,
			}, &resp)
			if err != nil {
				return err
			}
			fmt.Printf("Promote service: %s\n", *service)
			if resp.Status != "" {
				fmt.Printf("%s\n", resp.Status)
			} else {
				fmt.Printf("[✓] Promotion of %s from %s to %s initialized\n", resp.Tag, resp.FromEnvironment, resp.ToEnvironment)
			}

			return nil
		},
	}
	command.Flags().StringVar(&environment, "env", "", "Environment to promote to (required)")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	command.MarkFlagRequired("env")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	return command
}

package command

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewRollback(client *httpinternal.Client, service *string) *cobra.Command {
	var environment, namespace string
	var command = &cobra.Command{
		Use:   "rollback",
		Short: `Rollback to the previous artifact in an environment.`,
		Long: `Rollback to the previous artifact in an environment.

The command will release the artifact running in an environment before the
current one, ie. rollback a single release.

Note that 'rollback' does not traverse futher than one release. This means that
if you perform to rollbacks on the same environment after each other, the latter
has no effect.`,
		Example: `Rollback to the previous artifact for service 'product' in environment 'dev':

  hamctl rollback --service product --env dev`,
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}
			var resp httpinternal.RollbackResponse
			path, err := client.URL("rollback")
			if err != nil {
				return err
			}
			err = client.Do(http.MethodPost, path, httpinternal.RollbackRequest{
				Service:        *service,
				Namespace:      namespace,
				Environment:    environment,
				CommitterName:  committerName,
				CommitterEmail: committerEmail,
			}, &resp)
			if err != nil {
				return err
			}
			if resp.Status != "" {
				fmt.Printf("%s\n", resp.Status)
			}
			fmt.Printf("[✓] Rollback of artifact '%s' initiated\n", resp.PreviousArtifactID)
			fmt.Printf("    Release of '%s' to '%s'\n", resp.NewArtifactID, resp.Environment)

			return nil
		},
	}
	command.Flags().StringVar(&environment, "env", "", "environment to release to (required)")
	command.MarkFlagRequired("env")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	return command
}

package command

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
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
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			committer, err := git.CommitterDetails()
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
				CommitterName:  committer.Name,
				CommitterEmail: committer.Email,
			}, &resp)
			if err != nil {
				return err
			}
			fmt.Printf("Rollback of service: %s\n", *service)
			if resp.Status != "" {
				fmt.Printf("%s\n", resp.Status)
			}
			fmt.Printf("[✓] Rollback of artifact '%s' initiated\n", resp.PreviousArtifactID)
			fmt.Printf("    Release of '%s' to '%s'\n", resp.NewArtifactID, resp.Environment)

			return nil
		},
	}
	command.Flags().StringVarP(&environment, "env", "e", "", "environment to release to (required)")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	return command
}

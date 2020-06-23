package command

import (
	"fmt"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/spf13/cobra"
)

func NewRollback(client *httpinternal.Client, service *string) *cobra.Command {
	var environment, namespace, artifactID string
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
			var currentRelease httpinternal.DescribeReleaseResponseRelease
			var rollbackTo *httpinternal.DescribeReleaseResponseRelease

			if artifactID == "" {
				releases, err := actions.ReleasesFromEnvironment(client, *service, environment, 2)
				if err != nil {
					return err
				}

				if len(releases) < 2 {
					return fmt.Errorf("can't do rollback, because there isn't a release to rollback to")
				}
				currentRelease = releases[0]
				rollbackTo = &releases[1]
			} else {
				releases, err := actions.ReleasesFromEnvironment(client, *service, environment, 10)
				if err != nil {
					return err
				}

				for _, release := range releases {
					if release.Artifact.ID != artifactID {
						continue
					}
					rollbackTo = &release
					break
				}

				if rollbackTo == nil {
					return fmt.Errorf("can't do rollback, because the artifact '%s' ins't found in the last 10 releases", artifactID)
				}
				currentRelease = releases[0]
			}
			fmt.Printf("Rollback of service: %s\n", *service)

			err := actions.ReleaseArtifactID(client, *service, environment, rollbackTo.Artifact.ID, intent.NewRollback(currentRelease.Artifact.ID))
			if err != nil {
				fmt.Printf("[X] Rollback of artifact '%s' failed\n", currentRelease.Artifact.ID)
				fmt.Printf("    Error:\n")
				fmt.Printf("    %s\n", err)
				return err
			}
			fmt.Printf("[✓] Rollback of artifact '%s' initiated\n", currentRelease.Artifact.ID)
			fmt.Printf("    Release of '%s' to '%s'\n", rollbackTo.Artifact.ID, environment)

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
	command.Flags().StringVarP(&artifactID, "artifact-id", "", "", "artifact to roll back to. Defaults to previous artifact")
	return command
}

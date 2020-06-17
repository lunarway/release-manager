package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewRelease(client *httpinternal.Client, service *string) *cobra.Command {
	var environment, branch, artifact string
	var command = &cobra.Command{
		Use:   "release",
		Short: `Release a specific artifact or latest artifact from a branch into a specific environment.`,
		Example: `Release artifact 'master-482c9d808e-3bf40478e5' from service 'product' into environment 'dev':

  hamctl release --service product --env dev --artifact master-482c9d808e-3bf40478e5

Release latest artifact from branch 'master' of service 'product' into environment 'dev':

  hamctl release --service product --env dev --branch master`,
		RunE: func(c *cobra.Command, args []string) error {
			switch {
			case branch != "" && artifact != "":
				return errors.New("--branch and --artifact cannot both be specificed")

			case branch == "" && artifact == "":
				return errors.New("--branch or --artifact is required")
			case branch != "":
				artifactID, err := actions.ArtifactIDFromBranch(client, *service, branch)
				if err != nil {
					return err
				}
				actions.ReleaseArtifactID(client, *service, environment, artifactID, intent.NewReleaseBranch(branch))
			case artifact != "":
				actions.ReleaseArtifactID(client, *service, environment, artifact, intent.NewReleaseArtifact())
			}

			return nil
		},
	}
	command.Flags().StringVarP(&environment, "env", "e", "", "environment to release to (required)")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	command.Flags().StringVarP(&branch, "branch", "b", "", "release latest artifact from this branch (mutually exclusive with --artifact)")
	completion.FlagAnnotation(command, "branch", "__hamctl_get_branches")
	command.Flags().StringVar(&artifact, "artifact", "", "release this artifact id (mutually exclusive with --branch)")
	return command
}

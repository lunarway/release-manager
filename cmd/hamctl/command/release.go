package command

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewRelease(client *client) *cobra.Command {
	var serviceName, environment, branch, artifact string
	var command = &cobra.Command{
		Use:   "release",
		Short: `Release a specific artifact or latest artifact from a branch into a specific environment.`,
		Example: `Release artifact 'master-482c9d808e-3bf40478e5' from service 'product' into environemnt 'dev':

  hamctl release --service product --env dev --artifact master-482c9d808e-3bf40478e5

Release latest artifact from branch 'master' of service 'product' into environemnt 'dev':

  hamctl release --service product --env dev --branch master`,
		RunE: func(c *cobra.Command, args []string) error {
			if branch != "" && artifact != "" {
				return errors.New("--branch and --artifact cannot both be specificed")
			}
			if branch == "" && artifact == "" {
				return errors.New("--branch or --artifact is required")
			}
			committerName, committerEmail, err := git.CommitterDetails()
			if err != nil {
				return err
			}
			var resp httpinternal.ReleaseResponse
			path, err := client.url("release")
			if err != nil {
				return err
			}
			err = client.req(http.MethodPost, path, httpinternal.ReleaseRequest{
				Service:        serviceName,
				Environment:    environment,
				Branch:         branch,
				ArtifactID:     artifact,
				CommitterName:  committerName,
				CommitterEmail: committerEmail,
			}, &resp)
			if err != nil {
				return err
			}
			if resp.Status != "" {
				fmt.Printf("%s\n", resp.Status)
			} else {
				fmt.Printf("[✓] Release of %s to %s initialized\n", resp.Tag, resp.ToEnvironment)
			}

			return nil
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "service from with to release into specified environment (required)")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&environment, "env", "", "environment to release to (required)")
	command.MarkFlagRequired("env")
	command.Flags().StringVar(&branch, "branch", "", "release latest artifact from this branch (mutually exclusive with --artifact)")
	command.Flags().StringVar(&artifact, "artifact", "", "release this artifact id (mutually exclusive with --branch)")
	return command
}

package command

import (
	"strings"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type LoggerFunc = func(string, ...interface{})

type ReleaseArtifactMultipleEnvironments interface {
	ReleaseArtifactIDMultipleEnvironments(service string, environments []string, artifactID string, intent intent.Intent) ([]actions.ReleaseResult, error)
}

func NewRelease(client *httpinternal.Client, service *string, logger LoggerFunc, releaseClient ReleaseArtifactMultipleEnvironments) *cobra.Command {
	var branch, artifact string
	var environments []string
	var command = &cobra.Command{
		Use:   "release",
		Short: `Release a specific artifact or latest artifact from a branch into a specific environment.`,
		Example: `Release artifact 'master-482c9d808e-3bf40478e5' from service 'product' into environment 'dev':

  hamctl release --service product --env dev --artifact master-482c9d808e-3bf40478e5

Release latest artifact from branch 'master' of service 'product' into environment 'dev':

  hamctl release --service product --env dev --branch master`,
		Args: cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			environments = trimEmptyValues(environments)
			if len(environments) == 0 {
				return errors.New("--env must contain at least one value")
			}
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
				logger("Release of service %s using branch %s\n", *service, branch)
				resps, err := releaseClient.ReleaseArtifactIDMultipleEnvironments(*service, environments, artifactID, intent.NewReleaseBranch(branch))
				if err != nil {
					return err
				}
				for _, resp := range resps {
					printReleaseResponse(logger, resp)
				}
				return nil
			case artifact != "":
				logger("Release of service: %s\n", *service)
				resps, err := releaseClient.ReleaseArtifactIDMultipleEnvironments(*service, environments, artifact, intent.NewReleaseArtifact())
				if err != nil {
					return err
				}
				for _, resp := range resps {
					printReleaseResponse(logger, resp)
				}
				return nil
			}

			return nil
		},
	}
	command.Flags().StringSliceVarP(&environments, "env", "e", nil, "Comma separated list of environments to release to (required)")
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

func trimEmptyValues(values []string) []string {
	var trimmed []string
	for _, v := range values {
		t := strings.TrimSpace(v)
		if t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return trimmed
}

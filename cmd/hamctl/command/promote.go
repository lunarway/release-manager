package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewPromote(client *httpinternal.Client, service *string) *cobra.Command {
	var toEnvironment, fromEnvironment, namespace string
	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			switch toEnvironment {
			case "dev":
				// TODO: Deploy master branch to dev
				if fromEnvironment == "" {
					fromEnvironment = "master"
				}
			case "staging":
				// TODO: Deploy dev branch to staging
				if fromEnvironment == "" {
					fromEnvironment = "dev"
				}
			case "prod":
				// TODO: Deploy staging branch to prod
				if fromEnvironment == "" {
					fromEnvironment = "staging"
				}
			}
			artifactID, err := actions.ArtifactIDFromEnvironment(client, *service, namespace, fromEnvironment)
			if err != nil {
				return err
			}

			if err != nil {
				return err
			}

			return actions.ReleaseArtifactID(client, *service, toEnvironment, artifactID, struct{}{})
		},
	}
	command.Flags().StringVarP(&toEnvironment, "env", "", "", "Alias for '--to-env' (deprecated)")
	command.Flags().StringVarP(&toEnvironment, "to-env", "e", "", "Environment to promote to (required)")
	command.Flags().StringVarP(&fromEnvironment, "from-env", "", "", "Environment to promote from")

	return command
}

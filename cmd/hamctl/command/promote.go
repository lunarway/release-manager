package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
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
			if fromEnvironment == "" {
				switch toEnvironment {
				case "dev":
					fromEnvironment = "master"
				case "staging":
					fromEnvironment = "dev"
				case "prod":
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

			return actions.ReleaseArtifactID(client, *service, toEnvironment, artifactID, intent.NewPromoteEnvironment(fromEnvironment))
		},
	}
	command.Flags().StringVarP(&toEnvironment, "env", "", "", "Alias for '--to-env' (deprecated)")
	command.Flags().StringVarP(&toEnvironment, "to-env", "e", "", "Environment to promote to (required)")
	command.Flags().StringVarP(&fromEnvironment, "from-env", "", "", "Environment to promote from")

	return command
}

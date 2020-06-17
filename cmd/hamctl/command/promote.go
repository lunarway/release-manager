package command

import (
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
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
			var artifactID string
			var err error
			if fromEnvironment == "master" {
				artifactID, err = actions.ArtifactIDFromBranch(client, *service, "master")
			} else {
				artifactID, err = actions.ArtifactIDFromEnvironment(client, *service, namespace, fromEnvironment)
			}
			if err != nil {
				return err
			}

			return actions.ReleaseArtifactID(client, *service, toEnvironment, artifactID, intent.NewPromoteEnvironment(fromEnvironment))
		},
	}
	command.Flags().StringVarP(&toEnvironment, "env", "e", "", "Environment to promote to (required)")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")

	command.Flags().StringVarP(&fromEnvironment, "from-env", "", "", "Environment to promote from")
	completion.FlagAnnotation(command, "from-env", "__hamctl_get_environments")

	command.MarkFlagRequired("env")

	return command
}

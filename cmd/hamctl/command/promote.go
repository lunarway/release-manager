package command

import (
	"fmt"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/spf13/cobra"
)

func NewPromote(client *httpinternal.Client, service *string, releaseClient ReleaseArtifact) *cobra.Command {
	var toEnvironment, fromEnvironment, namespace string
	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		Args:  cobra.ExactArgs(0),
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
				case "prod":
					fromEnvironment = "dev"
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

			fmt.Printf("Promote of service: %s\n", *service)
			resp, err := releaseClient.ReleaseArtifactID(*service, toEnvironment, artifactID, intent.NewPromoteEnvironment(fromEnvironment))
			if err != nil {
				return err
			}
			printReleaseResponse(func(s string, i ...interface{}) {
				fmt.Printf(s, i...)
			}, resp)
			return nil
		},
	}
	command.Flags().StringVarP(&toEnvironment, "env", "e", "", "Environment to promote to (required)")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")

	command.Flags().StringVarP(&fromEnvironment, "from-env", "", "", "Environment to promote from")
	completion.FlagAnnotation(command, "from-env", "__hamctl_get_environments")

	command.MarkFlagRequired("env")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")

	return command
}

func printReleaseResponse(logger LoggerFunc, resp actions.ReleaseResult) {
	switch {
	case resp.Error != nil:
		logger("[X] %s\n", resp.Error.Error())
	case resp.Response.Status != "":
		logger("[✓] %s\n", resp.Response.Status)
	default:
		logger("[✓] Release of %s to %s initialized\n", resp.Response.Tag, resp.Response.ToEnvironment)
	}
}

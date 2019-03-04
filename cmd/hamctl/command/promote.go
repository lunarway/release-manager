package command

import (
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/spf13/cobra"
)

func NewPromote(options *Options) *cobra.Command {
	var serviceName, environment, configRepo, artifactFileName string

	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		RunE: func(c *cobra.Command, args []string) error {
			return flow.Promote(configRepo, artifactFileName, serviceName, environment)
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "Service to promote to specified environment (required)")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&environment, "env", "", "Environment to promote to (required)")
	command.MarkFlagRequired("env")
	command.Flags().StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "Kubernetes cluster configuration repository.")
	command.Flags().StringVar(&artifactFileName, "file", "artifact.json", "")
	return command
}

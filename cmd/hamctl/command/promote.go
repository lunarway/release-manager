package command

import (
	"context"
	"fmt"

	gengrpc "github.com/lunarway/release-manager/generated/grpc"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func NewPromote(options *Options) *cobra.Command {
	var serviceName, environment, configRepo, artifactFileName string
	var command = &cobra.Command{
		Use:   "promote",
		Short: "Promote a service to a specific environment following promoting conventions.",
		RunE: func(c *cobra.Command, args []string) error {
			conn, err := grpc.Dial(options.grpcAddress, grpc.WithInsecure())
			if err != nil {
				return err
			}
			defer conn.Close()
			client := gengrpc.NewReleaseManagerClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), options.grpcTimeout)
			defer cancel()
			r, err := client.Promote(ctx, &gengrpc.PromoteRequest{
				Service:     serviceName,
				Environment: environment,
			})
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", r.Status)
			return nil
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

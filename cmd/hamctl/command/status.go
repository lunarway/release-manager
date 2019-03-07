package command

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	gengrpc "github.com/lunarway/release-manager/generated/grpc"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func NewStatus(options *Options) *cobra.Command {
	var serviceName, configRepo, artifactFileName string
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		RunE: func(c *cobra.Command, args []string) error {
			conn, err := grpc.Dial(options.grpcAddress, grpc.WithInsecure())
			if err != nil {
				return err
			}
			defer conn.Close()
			client := gengrpc.NewReleaseManagerClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), options.grpcTimeout)
			defer cancel()
			r, err := client.Status(ctx, &gengrpc.StatusRequest{
				Service: serviceName,
			})
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			color.Green("k8s.dev.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n\n", r.Dev.Tag, r.Dev.Author, r.Dev.Committer, r.Dev.Message, time.Unix(r.Dev.Date, 0))
			color.Green("k8s.staging.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n\n", r.Staging.Tag, r.Staging.Author, r.Staging.Committer, r.Staging.Message, time.Unix(r.Staging.Date, 0))
			color.Green("kubernetes.prod.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n\n", r.Prod.Tag, r.Prod.Author, r.Prod.Committer, r.Prod.Message, time.Unix(r.Prod.Date, 0))
			return nil
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "Service to promote to specified environment (required)")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "Kubernetes cluster configuration repository.")
	command.Flags().StringVar(&artifactFileName, "file", "artifact.json", "")
	return command
}

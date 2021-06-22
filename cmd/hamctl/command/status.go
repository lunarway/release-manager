package command

import (
	"fmt"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/color"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/status"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/spf13/cobra"
)

func NewStatus(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var namespace string
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		Args:  cobra.ExactArgs(0),
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			resp, err := client.Status.GetStatus(status.NewGetStatusParams().WithNamespace(&namespace).WithService(*service), *clientAuth)
			if err != nil {
				return err
			}
			if !someManaged(resp.Payload.Dev, resp.Payload.Staging, resp.Payload.Prod) {
				if resp.Payload.DefaultNamespaces {
					fmt.Printf("Using default namespaces. ")
				}
				fmt.Printf("Are you setting the right namespace?\n")
			}
			fmt.Printf("Status for service: %s\n", *service)
			fmt.Printf("\n")
			color.Green("dev:\n")
			printStatus(resp.Payload.Dev)

			color.Green("staging:\n")
			printStatus(resp.Payload.Staging)

			color.Green("prod:\n")
			printStatus(resp.Payload.Prod)
			return nil
		},
	}
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	return command
}

// someManaged returns true if any of provided environments are managed.
func someManaged(envs ...*models.EnvironmentStatus) bool {
	for _, e := range envs {
		if e != nil && e.Tag != "" {
			return true
		}
	}
	return false
}
func Time(epoch int64) time.Time {
	return time.Unix(epoch/1000, 0)
}

func printStatus(e *models.EnvironmentStatus) {
	if e == nil {
		return
	}
	if e.Tag == "" {
		fmt.Printf("  Not managed by the release-manager\n\n")
		return
	}
	fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n  Link: %s\n  Vulnerabilities: %d high, %d medium, %d low\n\n", e.Tag, e.Author, e.Committer, e.Message, Time(e.Date), e.BuildURL, e.HighVulnerabilities, e.MediumVulnerabilities, e.LowVulnerabilities)
}

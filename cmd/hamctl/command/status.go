package command

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/lunarway/color"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewStatus(client *httpinternal.Client, service *string) *cobra.Command {
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
			var resp httpinternal.StatusResponse
			params := url.Values{}
			params.Add("service", *service)
			if namespace != "" {
				params.Add("namespace", namespace)
			}
			path, err := client.URLWithQuery("status", params)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodGet, path, nil, &resp)

			if err != nil {
				return err
			}
			if !someManaged(resp.Dev, resp.Staging, resp.Prod) {
				if resp.DefaultNamespaces {
					fmt.Printf("Using default namespaces. ")
				}
				fmt.Printf("Are you setting the right namespace?\n")
			}
			fmt.Printf("Status for service: %s\n", *service)
			fmt.Printf("\n")
			color.Green("dev:\n")
			printStatus(resp.Dev)

			color.Green("staging:\n")
			printStatus(resp.Staging)

			color.Green("prod:\n")
			printStatus(resp.Prod)
			return nil
		},
	}
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	return command
}

// someManaged returns true if any of provided environments are managed.
func someManaged(envs ...*httpinternal.Environment) bool {
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

func printStatus(e *httpinternal.Environment) {
	if e == nil {
		return
	}
	if e.Tag == "" {
		fmt.Printf("  Not managed by the release-manager\n\n")
		return
	}
	fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n  Link: %s\n  Vulnerabilities: %d high, %d medium, %d low\n\n", e.Tag, e.Author, e.Committer, e.Message, Time(e.Date), e.BuildUrl, e.HighVulnerabilities, e.MediumVulnerabilities, e.LowVulnerabilities)
}

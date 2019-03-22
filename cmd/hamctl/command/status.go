package command

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/lunarway/color"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewStatus(client *client) *cobra.Command {
	var serviceName, configRepo, artifactFileName string
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.StatusResponse
			params := url.Values{}
			params.Add("service", serviceName)
			path, err := client.urlWithQuery("status", params)
			if err != nil {
				return err
			}
			err = client.req(http.MethodGet, path, nil, &resp)

			if err != nil {
				return err
			}
			fmt.Printf("\n")
			color.Green("k8s.dev.lunarway.com\n")
			printStatus(resp.Dev)

			color.Green("k8s.staging.lunarway.com\n")
			printStatus(resp.Staging)

			color.Green("kubernetes.prod.lunarway.com\n")
			printStatus(resp.Prod)
			return nil
		},
	}
	command.Flags().StringVar(&serviceName, "service", "", "service to output current status for")
	command.MarkFlagRequired("service")
	command.Flags().StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "Kubernetes cluster configuration repository.")
	command.Flags().StringVar(&artifactFileName, "file", "artifact.json", "")
	return command
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

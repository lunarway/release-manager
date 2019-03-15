package command

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lunarway/color"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewStatus(options *Options) *cobra.Command {
	var serviceName, configRepo, artifactFileName string
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		RunE: func(c *cobra.Command, args []string) error {
			url := options.httpBaseURL + "/status?service=" + serviceName

			client := &http.Client{
				Timeout: options.httpTimeout,
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+options.authToken)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			decoder := json.NewDecoder(resp.Body)
			var r httpinternal.StatusResponse

			err = decoder.Decode(&r)
			if err != nil {
				return err
			}
			fmt.Printf("\n")
			color.Green("k8s.dev.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n  Link: %s\n  Vulnerabilities: %d high, %d medium, %d low\n\n", r.Dev.Tag, r.Dev.Author, r.Dev.Committer, r.Dev.Message, Time(r.Dev.Date), r.Dev.BuildUrl, r.Dev.HighVulnerabilities, r.Dev.MediumVulnerabilities, r.Dev.LowVulnerabilities)
			color.Green("k8s.staging.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n  Link: %s\n  Vulnerabilities: %d high, %d medium, %d low\n\n", r.Staging.Tag, r.Staging.Author, r.Staging.Committer, r.Staging.Message, Time(r.Staging.Date), r.Staging.BuildUrl, r.Staging.HighVulnerabilities, r.Staging.MediumVulnerabilities, r.Staging.LowVulnerabilities)
			color.Green("kubernetes.prod.lunarway.com\n")
			fmt.Printf("  Tag: %s\n  Author: %s\n  Committer: %s\n  Message: %s\n  Date: %s\n  Link: %s\n  Vulnerabilities: %d high, %d medium, %d low\n\n", r.Prod.Tag, r.Prod.Author, r.Prod.Committer, r.Prod.Message, Time(r.Prod.Date), r.Prod.BuildUrl, r.Prod.HighVulnerabilities, r.Prod.MediumVulnerabilities, r.Prod.LowVulnerabilities)
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

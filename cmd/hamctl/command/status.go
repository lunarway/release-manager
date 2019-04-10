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

func NewStatus(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.StatusResponse
			params := url.Values{}
			params.Add("service", *service)
			path, err := client.URLWithQuery("status", params)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodGet, path, nil, &resp)

			if err != nil {
				return err
			}
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

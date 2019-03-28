package policy

import (
	"fmt"
	"net/http"
	"net/url"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewList(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: "List current policies",
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.ListPoliciesResponse
			params := url.Values{}
			params.Add("service", *service)
			path, err := client.URLWithQuery("policy", params)
			if err != nil {
				return err
			}
			err = client.Req(http.MethodGet, path, nil, &resp)
			if err != nil {
				responseErr, ok := err.(*httpinternal.ErrorResponse)
				if !ok || responseErr.Status != http.StatusNotFound {
					return err
				}
				fmt.Printf("No policies exist for service\n")
				return nil
			}
			fmt.Printf("Policies for service %s\n", resp.Service)
			fmt.Println()
			if len(resp.AutoReleases) != 0 {
				fmt.Printf("Auto-releases:\n")
				fmt.Println()
				maxBranchLen := maxLen(resp.AutoReleases, func(p httpinternal.AutoReleasePolicy) string {
					return p.Branch
				})
				maxEnvLen := maxLen(resp.AutoReleases, func(p httpinternal.AutoReleasePolicy) string {
					return p.Environment
				})
				maxIDLen := maxLen(resp.AutoReleases, func(p httpinternal.AutoReleasePolicy) string {
					return p.ID
				})
				formatString := fmt.Sprintf("%%-%ds     %%-%ds     %%-%ds\n", maxBranchLen, maxEnvLen, maxIDLen)
				fmt.Printf(formatString, "BRANCH", "ENV", "ID")

				for _, p := range resp.AutoReleases {
					fmt.Printf(formatString, p.Branch, p.Environment, p.ID)
				}
			}
			return nil
		},
	}
	return command
}

func maxLen(policies []httpinternal.AutoReleasePolicy, f func(httpinternal.AutoReleasePolicy) string) int {
	longest := 0
	for _, p := range policies {
		str := f(p)
		if len(str) > longest {
			longest = len(str)
		}
	}
	return longest
}

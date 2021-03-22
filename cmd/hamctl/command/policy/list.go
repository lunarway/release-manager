package policy

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewList(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: "List current policies",
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.ListPoliciesResponse
			params := url.Values{}
			params.Add("service", *service)
			path, err := client.URLWithQuery(path, params)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodGet, path, nil, &resp)
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
			printAutoReleasePolicies(resp.AutoReleases)
			fmt.Println()
			printBranchRestrictionPolicies(resp.BranchRestrictions)
			return nil
		},
	}
	return command
}

func printAutoReleasePolicies(autoReleases []httpinternal.AutoReleasePolicy) {
	if len(autoReleases) == 0 {
		return
	}
	fmt.Printf("Auto-releases:\n")
	fmt.Println()
	maxBranchLen := maxLen(autoReleases, func(i int) string {
		return autoReleases[i].Branch
	})
	maxEnvLen := maxLen(autoReleases, func(i int) string {
		return autoReleases[i].Environment
	})
	maxIDLen := maxLen(autoReleases, func(i int) string {
		return autoReleases[i].ID
	})
	formatString := fmt.Sprintf("%%-%ds     %%-%ds     %%-%ds\n", maxEnvLen, maxBranchLen, maxIDLen)
	fmt.Printf(formatString, "ENV", "BRANCH", "ID")

	for _, p := range autoReleases {
		fmt.Printf(formatString, p.Environment, p.Branch, p.ID)
	}
}

func printBranchRestrictionPolicies(policies []httpinternal.BranchRestrictionPolicy) {
	if len(policies) == 0 {
		return
	}
	fmt.Printf("Branch restrictions:\n")
	fmt.Println()
	maxBranchLen := maxLen(policies, func(i int) string {
		return policies[i].BranchRegex
	})
	maxEnvLen := maxLen(policies, func(i int) string {
		return policies[i].Environment
	})
	maxIDLen := maxLen(policies, func(i int) string {
		return policies[i].ID
	})
	formatString := fmt.Sprintf("%%-%ds     %%-%ds     %%-%ds\n", maxEnvLen, maxBranchLen, maxIDLen)
	fmt.Printf(formatString, "ENV", "REGEX", "ID")

	for _, p := range policies {
		fmt.Printf(formatString, p.Environment, p.BranchRegex, p.ID)
	}
}

// maxLen returns the maximum length of the string returned by f in slice
// values.
func maxLen(values interface{}, f func(int) string) int {
	valuesType := reflect.TypeOf(values).Kind()
	if valuesType != reflect.Slice {
		panic("maxLen only works on slices")
	}

	s := reflect.ValueOf(values)
	longest := 0
	for i := 0; i < s.Len(); i++ {
		str := f(i)
		if len(str) > longest {
			longest = len(str)
		}
	}
	return longest
}

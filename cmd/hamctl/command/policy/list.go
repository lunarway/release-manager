package policy

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/policies"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/spf13/cobra"
)

func NewList(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: "List current policies",
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			resp, err := client.Policies.GetPolicies(policies.NewGetPoliciesParams().WithService(*service), *clientAuth)
			if err != nil {
				responseErr, ok := err.(*policies.GetPoliciesNotFound)
				if !ok || responseErr.Payload.Status != http.StatusNotFound {
					return err
				}
				fmt.Printf("No policies exist for service\n")
				return nil
			}
			fmt.Printf("Policies for service %s\n", resp.Payload.Service)
			fmt.Println()
			printAutoReleasePolicies(resp.Payload.AutoReleases)
			fmt.Println()
			printBranchRestrictionPolicies(resp.Payload.BranchRestrictions)
			return nil
		},
	}
	return command
}

func printAutoReleasePolicies(autoReleases []*models.GetPoliciesResponseAutoReleasesItems0) {
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

func printBranchRestrictionPolicies(policies []*models.GetPoliciesResponseBranchRestrictionsItems0) {
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

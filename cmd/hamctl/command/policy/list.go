package policy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"

	"github.com/lunarway/release-manager/cmd/hamctl/template"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

var listPoliciesTemplate = `Policies for service {{ .Service }}

{{ if ne (len .AutoReleases) 0 -}}
Auto-releases:
{{ $columnFormat := printf "%%-%ds     %%-%ds     %%-%ds" .AutoReleaseBranchMaxLen .AutoReleaseEnvMaxLen .AutoReleaseIDMaxLen }}
{{ printf $columnFormat "ENV" "BRANCH" "ID" }}
{{ range $k, $v := .AutoReleases -}}
{{ printf $columnFormat .Environment .Branch .ID }}
{{ end }}
{{ end -}}
{{ if ne (len .BranchRestrictions) 0 -}}
Branch restrictions:
{{ $columnFormat := printf "%%-%ds     %%-%ds     %%-%ds" .BranchRestrictionsEnvMaxLen .BranchRestrictionsBranchRegexMaxLen .BranchRestrictionsIDMaxLen }}
{{ printf $columnFormat "ENV" "REGEX" "ID" }}
{{ range $k, $v := .BranchRestrictions -}}
{{ printf $columnFormat .Environment .BranchRegex .ID }}
{{ end -}}
{{ end -}}
`

type listPoliciesData struct {
	Service                             string
	AutoReleases                        []listPoliciesDataAutoRelease
	AutoReleaseBranchMaxLen             int
	AutoReleaseEnvMaxLen                int
	AutoReleaseIDMaxLen                 int
	BranchRestrictions                  []listPoliciesDataBranchRestriction
	BranchRestrictionsBranchRegexMaxLen int
	BranchRestrictionsEnvMaxLen         int
	BranchRestrictionsIDMaxLen          int
}

type listPoliciesDataAutoRelease struct {
	Environment string
	Branch      string
	ID          string
}

type listPoliciesDataBranchRestriction struct {
	Environment string
	BranchRegex string
	ID          string
}

func templateListPolicies(dest io.Writer, data listPoliciesData) error {
	return template.Output(dest, "describeArtifact", listPoliciesTemplate, data)
}

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
			err = templateListPolicies(os.Stdout, mapListResponseToTemplate(resp))
			if err != nil {
				return err
			}
			return nil
		},
	}
	return command
}

func mapListResponseToTemplate(resp httpinternal.ListPoliciesResponse) listPoliciesData {
	var autoReleases []listPoliciesDataAutoRelease
	for _, r := range resp.AutoReleases {
		autoReleases = append(autoReleases, listPoliciesDataAutoRelease{
			ID:          r.ID,
			Environment: r.Environment,
			Branch:      r.Environment,
		})
	}

	var branchRestriction []listPoliciesDataBranchRestriction
	for _, b := range resp.BranchRestrictions {
		branchRestriction = append(branchRestriction, listPoliciesDataBranchRestriction{
			Environment: b.Environment,
			BranchRegex: b.BranchRegex,
			ID:          b.ID,
		})
	}

	return listPoliciesData{
		Service: resp.Service,

		AutoReleases: autoReleases,
		AutoReleaseBranchMaxLen: maxLen(autoReleases, func(i int) string {
			return autoReleases[i].Branch
		}),
		AutoReleaseEnvMaxLen: maxLen(autoReleases, func(i int) string {
			return autoReleases[i].Environment
		}),
		AutoReleaseIDMaxLen: maxLen(autoReleases, func(i int) string {
			return autoReleases[i].ID
		}),

		BranchRestrictions: branchRestriction,
		BranchRestrictionsBranchRegexMaxLen: maxLen(branchRestriction, func(i int) string {
			return branchRestriction[i].BranchRegex
		}),
		BranchRestrictionsEnvMaxLen: maxLen(branchRestriction, func(i int) string {
			return branchRestriction[i].Environment
		}),
		BranchRestrictionsIDMaxLen: maxLen(branchRestriction, func(i int) string {
			return branchRestriction[i].ID
		}),
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

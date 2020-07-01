package command

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewDescribe(client *httpinternal.Client, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "describe",
		Short: "Show details of resources controlled by the release manager.",
		Example: `Get details about available artifacts for a service:

	hamctl describe artifact --service product

Get details about the current release of product in the dev environment:

  hamctl describe release --service product --env dev`,
	}
	command.AddCommand(newDescribeRelease(client, service))
	command.AddCommand(newDescribeArtifact(client, service))
	return command
}

var describeReleaseDefaultTemplate = `Service: {{ .Service }}
Environment: {{ .Environment }}
{{ range $k, $v := .Releases }}
 - Artifact: {{ .Artifact.ID }}
   {{ if ne (len .Artifact.Namespace) 0 -}}
   Namespace:  {{ .Artifact.Namespace }}
   {{ end -}}
   Artifact from: {{ .Artifact.CI.End.Format "2006-01-02 15:04:03" }}
   Artifact by: {{ .Artifact.Application.CommitterName }} ({{ .Artifact.Application.CommitterEmail }})
   Released at: {{ .ReleasedAt.Format "2006-01-02 15:04:03" }}
   Released by: {{ .ReleasedByName }} ({{ .ReleasedByEmail }})
   Commit: {{ .Artifact.Application.URL }}
   Message: {{ .Artifact.Application.Message }}
   Intent: {{ .Intent | printIntent }}
{{ end }}
`

func newDescribeRelease(client *httpinternal.Client, service *string) *cobra.Command {
	var environment, namespace, template string
	var count int
	var command = &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases"},
		Short:   "Show details about a release.",
		Example: `Get details about the current release of product in the dev environment:

	hamctl describe release --service product --env dev

Format the output with a custom template:

	hamctl describe release --service product --env dev --template '{{ .Service }}'`,
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			releasesResponse, err := actions.ReleasesFromEnvironment(client, *service, environment, count)
			if len(template) == 0 {
				template = describeReleaseDefaultTemplate
			}
			err = templateOutput(os.Stdout, "describeRelease", template, releasesResponse)
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVarP(&environment, "env", "e", "", "environment to promote to (required)")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("env")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	command.Flags().StringVarP(&template, "template", "", "", "template string to format the output. The format is Go templates (http://golang.org/pkg/text/template/#pkg-overview). Available data structure is an 'http.DescribeReleaseResponse' struct.")
	command.Flags().IntVarP(&count, "count", "c", 1, "number of releases to describe (default 1)")
	return command
}

var describeArtifactDefaultTemplate = `Latest artifacts for service: {{ .Service }}

{{ rightPad "Date" 21 }}{{ rightPad "Artifact" 30 }}Message
{{ range $k, $v := .Artifacts -}}
{{ rightPad (.CI.End.Format "2006-01-02 15:04:03") 21 }}{{ rightPad .ID 30 }}{{ .Application.Message }}
{{ end -}}
`

func newDescribeArtifact(client *httpinternal.Client, service *string) *cobra.Command {
	var count int
	var template string
	var command = &cobra.Command{
		Use:     "artifact",
		Aliases: []string{"artifacts"},
		Short:   "Show details about an artifact.",
		Example: `Get details about available artifacts for a service:

	hamctl describe artifact --service product

Get details about the latest 5 artifacts for a service:

	hamctl describe artifact --service product --count 5

Format the output with a custom template:

	hamctl describe artifact --service product --template '{{ .Service }}'`,
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.DescribeArtifactResponse
			params := url.Values{}
			params.Add("count", fmt.Sprintf("%d", count))
			path, err := client.URLWithQuery(fmt.Sprintf("describe/artifact/%s", *service), params)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodGet, path, nil, &resp)
			if err != nil {
				return err
			}
			if len(template) == 0 {
				template = describeArtifactDefaultTemplate
			}
			err = templateOutput(os.Stdout, "describeArtifact", template, resp)
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().IntVar(&count, "count", 5, "Number of artifacts to return sorted by latest")
	command.Flags().StringVarP(&template, "template", "", "", "template string to format the output. The format is Go templates (http://golang.org/pkg/text/template/#pkg-overview). Available data structure is an 'http.DescribeArtifactResponse' struct.")
	return command
}

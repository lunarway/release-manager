package command

import (
	"os"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/status"
	"github.com/spf13/cobra"
)

func NewDescribe(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "describe",
		Short: "Show details of resources controlled by the release manager.",
		Example: `Get details about available artifacts for a service:

	hamctl describe artifact --service product

Get details about the current release of product in the dev environment:

  hamctl describe release --service product --env dev`,
	}
	command.AddCommand(newDescribeRelease(client, clientAuth, service))
	command.AddCommand(newDescribeArtifact(client, clientAuth, service))
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

func newDescribeRelease(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var environment, namespace, template string
	var count int64
	var command = &cobra.Command{
		Use:     "release",
		Aliases: []string{"releases"},
		Short:   "Show details about a release.",
		Example: `Get details about the current release of product in the dev environment:

	hamctl describe release --service product --env dev

Format the output with a custom template:

	hamctl describe release --service product --env dev --template '{{ .Service }}'`,
		Args: cobra.ExactArgs(0),
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			releasesResponse, err := client.Status.GetDescribeReleaseServiceEnvironment(
				status.NewGetDescribeReleaseServiceEnvironmentParams().
					WithCount(&count).
					WithService(*service).
					WithEnvironment(environment),
				*clientAuth)
			if err != nil {
				return err
			}
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
	command.Flags().Int64VarP(&count, "count", "c", 1, "number of releases to describe (default 1)")
	return command
}

var describeArtifactDefaultTemplate = `Latest artifacts for service: {{ .Service }}

{{ rightPad "Date" 21 }}{{ rightPad "Artifact" 30 }}Message
{{ range $k, $v := .Artifacts -}}
{{ rightPad (.CI.End.Format "2006-01-02 15:04:03") 21 }}{{ rightPad .ID 30 }}{{ .Application.Message }}
{{ end -}}
`

func newDescribeArtifact(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service *string) *cobra.Command {
	var count int64
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
		Args: cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			resp, err := client.Status.GetDescribeArtifactService(
				status.NewGetDescribeArtifactServiceParams().
					WithService(*service).WithCount(&count),
				*clientAuth)
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
	command.Flags().Int64Var(&count, "count", 5, "Number of artifacts to return sorted by latest")
	command.Flags().StringVarP(&template, "template", "", "", "template string to format the output. The format is Go templates (http://golang.org/pkg/text/template/#pkg-overview). Available data structure is an 'http.DescribeArtifactResponse' struct.")
	return command
}

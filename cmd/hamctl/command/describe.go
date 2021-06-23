package command

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/cmd/hamctl/template"
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
 - Artifact: {{ .ArtifactID }}
   {{ if ne (len .Namespace) 0 -}}
   Namespace:  {{ .Namespace }}
   {{ end -}}
   Artifact from: {{ .ArtifactFrom.Format dateFormat }}
   Artifact by: {{ .CommitterName }} ({{ .CommitterEmail }})
   Released at: {{ .ReleasedAt.Format dateFormat }}
   Released by: {{ .ReleasedByName }} ({{ .ReleasedByEmail }})
   Commit: {{ .CommitURL }}
   Message: {{ .CommitMessage }}
   Intent: {{ .Intent }}
{{ end }}
`

type describeReleaseData struct {
	Service     string
	Environment string
	Releases    []describeReleaseDataRelease
}

type describeReleaseDataRelease struct {
	ArtifactID      string
	Namespace       string
	ArtifactFrom    time.Time
	CommitterName   string
	CommitterEmail  string
	ReleasedAt      time.Time
	ReleasedByName  string
	ReleasedByEmail string
	CommitURL       string
	CommitMessage   string
	Intent          string
}

func templateDescribeRelease(dest io.Writer, templateText string, data describeReleaseData) error {
	return template.Output(dest, "describeRelease", templateText, data)
}

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
		Args: cobra.ExactArgs(0),
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			releasesResponse, err := actions.ReleasesFromEnvironment(client, *service, environment, count)
			if err != nil {
				return err
			}
			if len(template) == 0 {
				template = describeReleaseDefaultTemplate
			}
			err = templateDescribeRelease(os.Stdout, template, mapReleaseResponseToTemplate(releasesResponse))
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

func mapReleaseResponseToTemplate(resp httpinternal.DescribeReleaseResponse) describeReleaseData {
	var releases []describeReleaseDataRelease
	for _, release := range resp.Releases {
		releases = append(releases, describeReleaseDataRelease{
			ArtifactID:      release.Artifact.ID,
			Namespace:       release.Artifact.Namespace,
			ArtifactFrom:    release.Artifact.CI.End,
			CommitterName:   release.Artifact.Application.CommitterName,
			CommitterEmail:  release.Artifact.Application.CommitterEmail,
			ReleasedAt:      release.ReleasedAt,
			ReleasedByName:  release.ReleasedByName,
			ReleasedByEmail: release.ReleasedByEmail,
			CommitURL:       release.Artifact.Application.URL,
			CommitMessage:   release.Artifact.Application.Message,
			Intent:          template.IntentString(release.Intent),
		})
	}
	d := describeReleaseData{
		Service:     resp.Service,
		Environment: resp.Environment,
		Releases:    releases,
	}

	return d
}

var describeArtifactDefaultTemplate = `Latest artifacts for service: {{ .Service }}

{{ rightPad "Date" 21 }}{{ rightPad "Artifact" 30 }}Message
{{ range $k, $v := .Artifacts -}}
{{ rightPad (.ArtifactFrom.Format dateFormat) 21 }}{{ rightPad .ArtifactID 30 }}{{ .CommitMessage }}
{{ end -}}
`

type describeArtifactData struct {
	Service   string
	Artifacts []describeArtifactDataArtifact
}

type describeArtifactDataArtifact struct {
	ArtifactID    string
	ArtifactFrom  time.Time
	CommitMessage string
}

func templateDescribeArtifact(dest io.Writer, templateText string, data describeArtifactData) error {
	return template.Output(dest, "describeArtifact", templateText, data)
}

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
		Args: cobra.ExactArgs(0),
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
			err = templateDescribeArtifact(os.Stdout, template, mapArtifactResponseToTemplate(resp))
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

func mapArtifactResponseToTemplate(resp httpinternal.DescribeArtifactResponse) describeArtifactData {
	var artifacts []describeArtifactDataArtifact
	for _, a := range resp.Artifacts {
		artifacts = append(artifacts, describeArtifactDataArtifact{
			ArtifactID:    a.ID,
			ArtifactFrom:  a.CI.End,
			CommitMessage: a.Application.Message,
		})
	}
	return describeArtifactData{
		Service:   resp.Service,
		Artifacts: artifacts,
	}
}

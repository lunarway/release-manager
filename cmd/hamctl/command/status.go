package command

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/cmd/hamctl/template"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func NewStatus(client *httpinternal.Client, service *string) *cobra.Command {
	var namespace string
	var command = &cobra.Command{
		Use:   "status",
		Short: "List the status of the environments",
		Args:  cobra.ExactArgs(0),
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			var resp httpinternal.StatusResponse
			params := url.Values{}
			params.Add("service", *service)
			if namespace != "" {
				params.Add("namespace", namespace)
			}
			path, err := client.URLWithQuery("status", params)
			if err != nil {
				return err
			}
			err = client.Do(http.MethodGet, path, nil, &resp)
			if err != nil {
				return err
			}

			err = templateStatus(os.Stdout, mapToStatusData(resp, *service))
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	return command
}

func mapToStatusData(resp httpinternal.StatusResponse, service string) statusData {
	return statusData{
		EnvironmentsManaged:    someManaged(resp.Dev, resp.Staging, resp.Prod),
		UsingDefaultNamespaces: resp.DefaultNamespaces,
		Service:                service,
		Environments: []statusDataEnvironment{
			mapEnvironment(resp.Dev, "dev"),
			mapEnvironment(resp.Staging, "staging"),
			mapEnvironment(resp.Prod, "prod"),
		},
	}
}

var statusTemplate = `Status for service {{ .Service }}
{{- if not .EnvironmentsManaged }}

No environments managed by release-manager.

{{ if .UsingDefaultNamespaces -}}
Using environment specific namespace, ie. dev, staging, prod.
{{ end -}}
Are you setting the right namespace?
{{- end -}}

{{ range .Environments }}

{{ .Environment }}:
{{- if eq (len .Tag) 0 }}
  Not managed by the release-manager
{{- else }}
  Tag: {{ .Tag }}
  Author: {{ .Author }}
  Committer: {{ .Committer }}
  Message: {{ .CommitMessage }}
  Date: {{ .Date.Format dateFormat }}
  Link: {{ .BuildURL }}
  Vulnerabilities: {{ .HighVulnerabilities }} high, {{ .MediumVulnerabilities }} medium, {{ .LowVulnerabilities }} low
{{- end -}}
{{- end }}
`

type statusData struct {
	EnvironmentsManaged    bool
	UsingDefaultNamespaces bool
	Service                string
	Environments           []statusDataEnvironment
}

type statusDataEnvironment struct {
	Environment           string
	Tag                   string
	Author                string
	Committer             string
	CommitMessage         string
	Date                  time.Time
	BuildURL              string
	HighVulnerabilities   int64
	MediumVulnerabilities int64
	LowVulnerabilities    int64
}

func templateStatus(dest io.Writer, data statusData) error {
	return template.Output(dest, "status", statusTemplate, data)
}

func mapEnvironment(env *httpinternal.Environment, name string) statusDataEnvironment {
	return statusDataEnvironment{
		Environment:           name,
		Tag:                   env.Tag,
		Author:                env.Author,
		Committer:             env.Committer,
		CommitMessage:         env.Message,
		Date:                  Time(env.Date),
		BuildURL:              env.BuildUrl,
		HighVulnerabilities:   env.HighVulnerabilities,
		MediumVulnerabilities: env.MediumVulnerabilities,
		LowVulnerabilities:    env.LowVulnerabilities,
	}
}

// someManaged returns true if any of provided environments are managed.
func someManaged(envs ...*httpinternal.Environment) bool {
	for _, e := range envs {
		if e != nil && e.Tag != "" {
			return true
		}
	}
	return false
}

func Time(epoch int64) time.Time {
	return time.Unix(epoch/1000, 0)
}

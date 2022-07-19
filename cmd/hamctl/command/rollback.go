package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/cmd/hamctl/template"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewRollback(
	client *httpinternal.Client,
	service *string,
	logger LoggerFunc,
	selectReleaseUI SelectRollbackRelease,
) *cobra.Command {
	var environment, namespace, artifactID string
	command := &cobra.Command{
		Use:   "rollback",
		Short: `Rollback to the previous artifact in an environment.`,
		Long: `Rollback to the previous artifact in an environment.

The command will release the artifact running in an environment before the
current one, ie. rollback a single release.

Note that 'rollback' does not traverse futher than one release. This means that
if you perform to rollbacks on the same environment after each other, the latter
has no effect.`,
		Example: `Rollback to the previous artifact for service 'product' in environment 'dev':

  hamctl rollback --service product --env dev`,
		Args: cobra.ExactArgs(0),
		PreRun: func(c *cobra.Command, args []string) {
			defaultShuttleString(shuttleSpecFromFile, &namespace, func(s *shuttleSpec) string {
				return s.Vars.K8S.Namespace
			})
		},
		RunE: func(c *cobra.Command, args []string) error {
			var currentRelease httpinternal.DescribeReleaseResponseRelease
			var rollbackTo *httpinternal.DescribeReleaseResponseRelease

			if artifactID == "" {
				releasesResponse, err := actions.ReleasesFromEnvironment(client, *service, environment, 3)
				if err != nil {
					return err
				}

				if len(releasesResponse.Releases) < 2 {
					return fmt.Errorf("can't do rollback, because there isn't a release to rollback to")
				}

				index, err := selectReleaseUI(environment, releasesResponse.Releases)
				if err != nil {
					logger("something went wrong select release a release")
					return err
				}

				currentRelease = releasesResponse.Releases[0]
				rollbackTo = &releasesResponse.Releases[index]
			} else {
				releasesResponse, err := actions.ReleasesFromEnvironment(client, *service, environment, 10)
				if err != nil {
					return err
				}

				for _, release := range releasesResponse.Releases {
					if release.Artifact.ID != artifactID {
						continue
					}
					rollbackTo = &release
					break
				}

				if rollbackTo == nil {
					return fmt.Errorf("can't do rollback, because the artifact '%s' isn't found in the last 10 releases", artifactID)
				}
				currentRelease = releasesResponse.Releases[0]
			}
			logger("[✓] Starting rollback of service %s to %s\n", *service, rollbackTo.Artifact.ID)

			resp, err := actions.ReleaseArtifactID(
				client,
				*service,
				environment,
				rollbackTo.Artifact.ID,
				intent.NewRollback(currentRelease.Artifact.ID),
			)
			if err != nil {
				logger("[X] Rollback of artifact '%s' failed\n", currentRelease.Artifact.ID)
				logger("    Error:\n")
				logger("    %s\n", err)
				return err
			}

			printReleaseResponse(func(s string, i ...interface{}) {
				logger(s, i...)
			}, resp)
			return nil
		},
	}
	command.Flags().StringVarP(&environment, "env", "e", "", "environment to release to (required)")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	command.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace the service is deployed to (defaults to env)")
	completion.FlagAnnotation(command, "namespace", "__hamctl_get_namespaces")
	command.Flags().
		StringVarP(&artifactID, "artifact", "", "", "artifact to roll back to. Defaults to previously released artifact for the environment")
	return command
}

type SelectRollbackRelease func(environment string, releases []httpinternal.DescribeReleaseResponseRelease) (int, error)

func SelectRollbackReleaseFunc(
	environment string,
	releases []httpinternal.DescribeReleaseResponseRelease,
) (int, error) {
	funcMap := promptui.FuncMap
	for name, f := range template.FuncMap() {
		funcMap[name] = f
	}
	rollbackInteractiveTemplates.FuncMap = funcMap

	items := mapToRollbackInteractiveTemplateData(releases)

	searcher := func(input string, index int) bool {
		release := items[index]
		name := strings.ToLower(fmt.Sprintf("#%d %s", release.ReleaseIndex, release.ArtifactID))
		input = strings.ToLower(input)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:             fmt.Sprintf("Which release to rollback '%s' to?", environment),
		Items:             items,
		Templates:         &rollbackInteractiveTemplates,
		Size:              10,
		Searcher:          searcher,
		StartInSearchMode: true,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return 0, err
	}

	return index, nil
}

var (
	rollbackInteractiveItemTemlpate = "#{{ .ReleaseIndex }} {{ .ArtifactID | cyan }}{{ if eq .ReleaseIndex 0 }} current release{{ end }} {{ .ReleasedAt | humanizeTime }} by {{ .ReleasedByEmail | blue }}"
	rollbackInteractiveTemplates    = promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> " + rollbackInteractiveItemTemlpate,
		Inactive: "   " + rollbackInteractiveItemTemlpate,
		Selected: "-> " + rollbackInteractiveItemTemlpate,
		Details: `
     {{ print "Release Details          " | bold | underline }}
     Release Number: {{ .ReleaseIndex }}
     Released at: {{ .ReleasedAt.Format dateFormat }}
     Released by: {{ .ReleasedByName }} ({{ .ReleasedByEmail }})
     Intent: {{ .Intent }}

     {{ print "Artifact Details          " | bold | underline }}
     Artifact: {{ .ArtifactID | cyan }}
     {{ if ne (len .Namespace) 0 -}}
     Namespace:  {{ .Namespace }}
     {{ end -}}
     Artifact from: {{ .ArtifactFrom.Format dateFormat }}
     Artifact by: {{ .CommitterName }} ({{ .CommitterEmail }})
     Commit: {{ .CommitURL }}
     Message: {{ .CommitMessage }}`,
	}
)

type rollbackInteractiveTemplatesData struct {
	ReleaseIndex    int
	ReleasedAt      time.Time
	ReleasedByName  string
	ReleasedByEmail string
	Intent          string
	ArtifactID      string
	Namespace       string
	ArtifactFrom    time.Time
	CommitterName   string
	CommitterEmail  string
	CommitURL       string
	CommitMessage   string
}

func mapToRollbackInteractiveTemplateData(
	resp []httpinternal.DescribeReleaseResponseRelease,
) []rollbackInteractiveTemplatesData {
	var d []rollbackInteractiveTemplatesData
	for _, r := range resp {
		d = append(d, rollbackInteractiveTemplatesData{
			ReleaseIndex:    r.ReleaseIndex,
			ReleasedAt:      r.ReleasedAt,
			ReleasedByName:  r.ReleasedByName,
			ReleasedByEmail: r.ReleasedByEmail,
			Intent:          template.IntentString(r.Intent),
			ArtifactID:      r.Artifact.ID,
			Namespace:       r.Artifact.Namespace,
			ArtifactFrom:    r.Artifact.CI.End,
			CommitterName:   r.Artifact.Application.CommitterName,
			CommitterEmail:  r.Artifact.Application.CommitterEmail,
			CommitURL:       r.Artifact.Application.URL,
			CommitMessage:   r.Artifact.Application.Message,
		})
	}
	return d
}

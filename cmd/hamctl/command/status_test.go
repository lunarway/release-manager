package command

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTemplateStatus(t *testing.T) {
	testCases := []struct {
		desc   string
		input  statusData
		output string
	}{
		{
			desc: "non managed",
			input: statusData{
				EnvironmentsManaged:    false,
				UsingDefaultNamespaces: false,
				Service:                "svc",
				Environments:           nil,
			},
			output: `Status for service svc

No environments managed by release-manager.

Are you setting the right namespace?
`,
		},
		{
			desc: "non managed with default namespaces",
			input: statusData{
				EnvironmentsManaged:    false,
				UsingDefaultNamespaces: true,
				Service:                "svc",
				Environments:           nil,
			},
			output: `Status for service svc

No environments managed by release-manager.

Using environment specific namespace, ie. dev, staging, prod.
Are you setting the right namespace?
`,
		},
		{
			desc: "single environment",
			input: statusData{
				EnvironmentsManaged:    true,
				UsingDefaultNamespaces: true,
				Service:                "svc",
				Environments: []statusDataEnvironment{
					{
						Environment:           "dev",
						Tag:                   "master-1234-5678",
						Author:                "John Doe",
						Committer:             "Jane Doe",
						CommitMessage:         "Useful bits",
						Date:                  time.Date(2021, time.June, 23, 8, 42, 22, 0, time.UTC),
						BuildURL:              "https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect",
						HighVulnerabilities:   1,
						MediumVulnerabilities: 2,
						LowVulnerabilities:    3,
					},
				},
			},
			output: `Status for service svc

dev:
  Tag: master-1234-5678
  Author: John Doe
  Committer: Jane Doe
  Message: Useful bits
  Date: 2021-06-23 08:42:22
  Link: https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect
  Vulnerabilities: 1 high, 2 medium, 3 low
`,
		},
		{
			desc: "multiple environments",
			input: statusData{
				EnvironmentsManaged:    true,
				UsingDefaultNamespaces: true,
				Service:                "svc",
				Environments: []statusDataEnvironment{
					{
						Environment:           "dev",
						Tag:                   "master-1234-5678",
						Author:                "John Doe",
						Committer:             "Jane Doe",
						CommitMessage:         "Useful bits",
						Date:                  time.Date(2021, time.June, 23, 8, 42, 22, 0, time.UTC),
						BuildURL:              "https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect",
						HighVulnerabilities:   1,
						MediumVulnerabilities: 2,
						LowVulnerabilities:    3,
					},
					{
						Environment:           "staging",
						Tag:                   "master-1234-5678",
						Author:                "John Doe",
						Committer:             "Jane Doe",
						CommitMessage:         "Useful bits",
						Date:                  time.Date(2021, time.June, 23, 8, 42, 22, 0, time.UTC),
						BuildURL:              "https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect",
						HighVulnerabilities:   1,
						MediumVulnerabilities: 2,
						LowVulnerabilities:    3,
					},
				},
			},
			output: `Status for service svc

dev:
  Tag: master-1234-5678
  Author: John Doe
  Committer: Jane Doe
  Message: Useful bits
  Date: 2021-06-23 08:42:22
  Link: https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect
  Vulnerabilities: 1 high, 2 medium, 3 low

staging:
  Tag: master-1234-5678
  Author: John Doe
  Committer: Jane Doe
  Message: Useful bits
  Date: 2021-06-23 08:42:22
  Link: https://jenkins.corp.com/job/github-lunarway/job/svc/job/master/105/display/redirect
  Vulnerabilities: 1 high, 2 medium, 3 low
`,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			var output bytes.Buffer
			err := templateStatus(&output, tC.input)
			require.NoError(t, err)
			require.Equal(t, tC.output, output.String())
		})
	}
}

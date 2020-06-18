package commitinfo

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseCommit(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage string
		commitInfo    ConventionalCommitInfo
		err           error
	}{
		{
			name:          "message with no description and field + no space",
			commitMessage: "[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar\nArtifact-created-by: Foo Bar <test@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar",
				Description: "",
				Fields: map[string]string{
					"Artifact-created-by": "Foo Bar <test@lunar.app>",
				},
			},

			err: nil,
		},
		{
			name:          "message with no description and field + space",
			commitMessage: "[test-service] artifact master-1234ds13g3-12s46g356g by test@lunar.app\n\nArtifact-created-by: Foo Bar <test@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[test-service] artifact master-1234ds13g3-12s46g356g by test@lunar.app",
				Description: "",
				Fields: map[string]string{
					"Artifact-created-by": "Foo Bar <test@lunar.app>",
				},
			},
			err: nil,
		},
		{
			name:          "only message",
			commitMessage: "[product] build something",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] build something",
				Description: "",
				Fields:      map[string]string{},
			},
			err: nil,
		},
		{
			name:          "message and multiple fields + no space",
			commitMessage: "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: map[string]string{
					"Artifact-created-by":  "Emil Ingerslev <eki@lunar.app>",
					"Artifact-released-by": "Bjørn Hald Sørensen <bso@lunar.app>",
				},
			},
			err: nil,
		},
		{
			name:          "message and multiple fields + space",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\n\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: map[string]string{
					"Artifact-created-by":  "Emil Ingerslev <eki@lunar.app>",
					"Artifact-released-by": "Bjørn Hald Sørensen <bso@lunar.app>",
				},
			},
			err: nil,
		},

		{
			name:          "message, description and multiple fields + no space",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\nSome description\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "Some description",
				Fields: map[string]string{
					"Artifact-created-by":  "Emil Ingerslev <eki@lunar.app>",
					"Artifact-released-by": "Bjørn Hald Sørensen <bso@lunar.app>",
				},
			},
		},
		{
			name:          "message, description and multiple fields + space",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\n\nSome description\n\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "Some description",
				Fields: map[string]string{
					"Artifact-created-by":  "Emil Ingerslev <eki@lunar.app>",
					"Artifact-released-by": "Bjørn Hald Sørensen <bso@lunar.app>",
				},
			},
			err: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseCommit(tc.commitMessage)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected")
		})
	}
}

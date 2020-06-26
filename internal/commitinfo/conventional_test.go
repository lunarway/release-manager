package commitinfo

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseConventionalCommit(t *testing.T) {
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
				Fields: []Field{
					NewField("Artifact-created-by", "Foo Bar <test@lunar.app>"),
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
				Fields: []Field{
					NewField("Artifact-created-by", "Foo Bar <test@lunar.app>"),
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
				Fields:      nil,
			},
			err: nil,
		},
		{
			name:          "message and multiple fields + no space",
			commitMessage: "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
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
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
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
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},
		{
			name:          "message, description and multiple fields + space",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\n\nSome description\n\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "Some description",
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
			err: nil,
		},
		{
			name:          "empty fields should parse just fine",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\n\nArtifact-created-by:\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", ""),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
			err: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseConventionalCommit(tc.commitMessage)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
				return
			} else {
				if !assert.NoError(t, err, "no output error expected") {
					return
				}
			}
			assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected")
		})
	}
}

func TestSetField(t *testing.T) {
	testCases := []struct {
		desc     string
		setup    []Field
		name     string
		value    string
		expected []Field
	}{
		{
			desc:  "set field with no current fields add field",
			setup: nil,
			name:  "Some",
			value: "value",
			expected: []Field{
				NewField("Some", "value"),
			},
		},
		{
			desc: "set field with matching current fields sets field",
			setup: []Field{
				NewField("Some", "old"),
			},
			name:  "Some",
			value: "new",
			expected: []Field{
				NewField("Some", "new"),
			},
		},
		{
			desc: "set field with not matching current fields adds field",
			setup: []Field{
				NewField("Some", "value"),
			},
			name:  "Another",
			value: "1337",
			expected: []Field{
				NewField("Some", "value"),
				NewField("Another", "1337"),
			},
		},
		{
			desc: "set field with matching current field sets value and keeps order",
			setup: []Field{
				NewField("Some", "old"),
				NewField("Another", "1337"),
			},
			name:  "Some",
			value: "new",
			expected: []Field{
				NewField("Some", "new"),
				NewField("Another", "1337"),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			cci := ConventionalCommitInfo{
				Fields: tC.setup,
			}
			cci.SetField(tC.name, tC.value)
			assert.Equal(t, tC.expected, cci.Fields, "expected fields does not match actual")
		})
	}
}

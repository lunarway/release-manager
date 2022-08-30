package commitinfo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConventionalCommit(t *testing.T) {
	tt := []struct {
		name                  string
		commitMessage         []string
		commitInfo            ConventionalCommitInfo
		expectedCommitMessage []string
	}{
		{
			name: "message with no description and field + no space",
			commitMessage: []string{
				"[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", "Foo Bar <test@lunar.app>"),
				},
			},
		},
		{
			name: "message with no description and field + space",
			commitMessage: []string{
				"[test-service] artifact master-1234ds13g3-12s46g356g by test@lunar.app",
				"",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[test-service] artifact master-1234ds13g3-12s46g356g by test@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", "Foo Bar <test@lunar.app>"),
				},
			},
		},
		{
			name: "only message",
			commitMessage: []string{
				"[product] build something",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] build something",
				Description: "",
				Fields:      nil,
			},
		},
		{
			name: "message and multiple fields + no space",
			commitMessage: []string{
				"[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},
		{
			name: "message and multiple fields + space",
			commitMessage: []string{
				"[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", "Emil Ingerslev <eki@lunar.app>"),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},

		{
			name: "message, description and multiple fields + no space",
			commitMessage: []string{
				"[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"Some description",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
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
			name: "message, description and multiple fields + space",
			commitMessage: []string{
				"[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Some description",
				"",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
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
			name: "empty fields should parse just fine",
			commitMessage: []string{
				"[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Artifact-created-by:",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				Description: "",
				Fields: []Field{
					NewField("Artifact-created-by", ""),
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},
		{
			name: "no message but description should parse just fine",
			commitMessage: []string{
				"",
				"",
				"Some description",
				"",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "",
				Description: "Some description",
				Fields: []Field{
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},
		{
			name: "only fields should parse just fine",
			commitMessage: []string{
				"",
				"",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: ConventionalCommitInfo{
				Message:     "",
				Description: "",
				Fields: []Field{
					NewField("Artifact-released-by", "Bjørn Hald Sørensen <bso@lunar.app>"),
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseConventionalCommit(strings.Join(tc.commitMessage, "\n"))
			assert.NoError(t, err, "no output error expected")
			assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected")
		})

		t.Run(tc.name+" and back", func(t *testing.T) {
			expectedCommitMessage := tc.commitMessage
			if tc.expectedCommitMessage == nil {
				expectedCommitMessage = tc.expectedCommitMessage
				return
			}
			assert.Equal(t, strings.Join(expectedCommitMessage, "\n"), tc.commitInfo.String())
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

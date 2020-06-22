package commitinfo

import (
	"strings"
	"testing"

	"github.com/lunarway/release-manager/internal/intent"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseCommitInfo(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage []string
		commitInfo    CommitInfo
		err           error
	}{
		{
			name: "artifact commit should not match",
			commitMessage: []string{
				"[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
			},
			err: ErrNoMatch,
		},
		{
			name: "release commit with no spacing should match",
			commitMessage: []string{
				"[staging/test-service] release master-1234ds13g3-12s46g356g by test@lunar.app",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "master-1234ds13g3-12s46g356g",
				ArtifactCreatedBy: NewPersonInfo("Foo Bar", "test@lunar.app"),
				Service:           "test-service",
				Environment:       "staging",
				Intent:            intent.NewReleaseArtifact(),
			},
		},
		{
			name: "not valid message",
			commitMessage: []string{
				"[product] build something",
			},
			err: errors.New("no match"),
		},
		{
			name: "release commit from product should match",
			commitMessage: []string{
				"[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "test-s3-push-f4440b4ccb-1ba3085aa7",
				Environment:       "dev",
				Service:           "product",
				ArtifactCreatedBy: NewPersonInfo("Emil Ingerslev", "eki@lunar.app"),
				ReleasedBy:        NewPersonInfo("Bjørn Hald Sørensen", "bso@lunar.app"),
				Intent:            intent.NewReleaseArtifact(),
			},
		},
		{
			name: "release commit with spacing should match",
			commitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "test-s3-push-f4440b4ccb-1ba3085aa7",
				Environment:       "prod",
				Service:           "product",
				ArtifactCreatedBy: NewPersonInfo("Emil Ingerslev", "eki@lunar.app"),
				ReleasedBy:        NewPersonInfo("Bjørn Hald Sørensen", "bso@lunar.app"),
				Intent:            intent.NewReleaseArtifact(),
			},
		},
		{
			name: "release with artifact release intent with should match",
			commitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Release-intent: ReleaseArtifact",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "test-s3-push-f4440b4ccb-1ba3085aa7",
				Environment:       "prod",
				Service:           "product",
				ArtifactCreatedBy: NewPersonInfo("Emil Ingerslev", "eki@lunar.app"),
				ReleasedBy:        NewPersonInfo("Bjørn Hald Sørensen", "bso@lunar.app"),
				Intent:            intent.NewReleaseArtifact(),
			},
		},
		{
			name: "release with branch release intent with should match",
			commitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app",
				"",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Release-intent: ReleaseBranch",
				"Release-branch: test-s3-push",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "test-s3-push-f4440b4ccb-1ba3085aa7",
				Environment:       "prod",
				Service:           "product",
				ArtifactCreatedBy: NewPersonInfo("Emil Ingerslev", "eki@lunar.app"),
				ReleasedBy:        NewPersonInfo("Bjørn Hald Sørensen", "bso@lunar.app"),
				Intent:            intent.NewReleaseBranch("test-s3-push"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseCommitInfo(strings.Join(tc.commitMessage, "\n"))
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected")
		})
	}
}

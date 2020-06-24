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
		name                 string
		commitMessage        []string
		commitInfo           CommitInfo
		err                  error
		correctCommitMessage []string
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
				"[staging/test-service] release master-1234ds13g3-12s46g356g by hest@lunar.app",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "master-1234ds13g3-12s46g356g",
				ArtifactCreatedBy: NewPersonInfo("Foo Bar", "test@lunar.app"),
				ReleasedBy:        NewPersonInfo("", "hest@lunar.app"),
				Service:           "test-service",
				Environment:       "staging",
				Intent:            intent.NewReleaseArtifact(),
			},
			correctCommitMessage: []string{
				"[staging/test-service] release master-1234ds13g3-12s46g356g by hest@lunar.app",
				"",
				"Service: test-service",
				"Environment: staging",
				"Artifact-ID: master-1234ds13g3-12s46g356g",
				"Artifact-released-by:  <hest@lunar.app>",
				"Artifact-created-by: Foo Bar <test@lunar.app>",
				"Release-intent: ReleaseArtifact",
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
			name: "release commit from product with intent should match",
			commitMessage: []string{
				"[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by bso@lunar.app",
				"",
				"Service: product",
				"Environment: dev",
				"Artifact-ID: test-s3-push-f4440b4ccb-1ba3085aa7",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Release-intent: ReleaseArtifact",
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
			name: "release with artifact release intent should match",
			commitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by bso@lunar.app",
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
			correctCommitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by bso@lunar.app",
				"",
				"Service: product",
				"Environment: prod",
				"Artifact-ID: test-s3-push-f4440b4ccb-1ba3085aa7",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Release-intent: ReleaseArtifact",
			},
		},
		{
			name: "release with branch release intent with should match",
			commitMessage: []string{
				"[prod/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by bso@lunar.app",
				"",
				"Service: product",
				"Environment: prod",
				"Artifact-ID: test-s3-push-f4440b4ccb-1ba3085aa7",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
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
		{
			name: "release with rollback release intent with should match",
			commitMessage: []string{
				"[prod/product] rollback test-s3-push-f4440b4ccb-1ba3085aa7 by bso@lunar.app",
				"",
				"Service: product",
				"Environment: prod",
				"Artifact-ID: test-s3-push-f4440b4ccb-1ba3085aa7",
				"Artifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
				"Artifact-created-by: Emil Ingerslev <eki@lunar.app>",
				"Release-intent: Rollback",
				"Release-rollback-of-artifact-id: test-s3-push-1337-1337",
			},
			commitInfo: CommitInfo{
				ArtifactID:        "test-s3-push-f4440b4ccb-1ba3085aa7",
				Environment:       "prod",
				Service:           "product",
				ArtifactCreatedBy: NewPersonInfo("Emil Ingerslev", "eki@lunar.app"),
				ReleasedBy:        NewPersonInfo("Bjørn Hald Sørensen", "bso@lunar.app"),
				Intent:            intent.NewRollback("test-s3-push-1337-1337"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseCommitInfo(strings.Join(tc.commitMessage, "\n"))
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
				return
			} else {
				if !assert.NoError(t, err, "no output error expected") {
					return
				}
			}
			if !assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected") {
				return
			}
		})
		if tc.err != nil {
			continue
		}
		t.Run(tc.name+" and back", func(t *testing.T) {
			actualMessage := tc.commitInfo.String()
			if tc.correctCommitMessage != nil {
				assert.Equal(t, strings.Join(tc.correctCommitMessage, "\n"), actualMessage, "commitInfo.String() does not match test.correctCommitMessage")
				return
			}
			assert.Equal(t, strings.Join(tc.commitMessage, "\n"), actualMessage, "commitInfo.String() does not match test.commitMessage")
		})
	}
}

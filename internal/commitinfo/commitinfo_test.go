package commitinfo

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestExtractInfoFromCommit(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage string
		commitInfo    CommitInfo
		err           error
	}{
		{
			name:          "exact values",
			commitMessage: "[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar\nArtifact-created-by: Foo Bar <test@lunar.app>",
			commitInfo: CommitInfo{
				ArtifactID:  "master-1234ds13g3-12s46g356g",
				AuthorEmail: "test@lunar.app",
				AuthorName:  "Foo Bar",
				Service:     "test-service",
			},
			err: nil,
		},
		{
			name:          "email as author",
			commitMessage: "[test-service] artifact master-1234ds13g3-12s46g356g by test@lunar.app\nArtifact-created-by: Foo Bar <test@lunar.app>",
			commitInfo: CommitInfo{
				ArtifactID:  "master-1234ds13g3-12s46g356g",
				AuthorEmail: "test@lunar.app",
				AuthorName:  "Foo Bar",
				Service:     "test-service",
			},
			err: nil,
		},
		{
			name:          "not valid message",
			commitMessage: "[product] build something",
			commitInfo:    CommitInfo{},
			err:           errors.New("no match"),
		},
		{
			name:          "release commit from product should not match",
			commitMessage: "[dev/product] release test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>\nArtifact-released-by: Bjørn Hald Sørensen <bso@lunar.app>",
			commitInfo:    CommitInfo{},
			err:           errors.New("no match"),
		},
		{
			name:          "artifact commit from product should match",
			commitMessage: "[product] artifact test-s3-push-f4440b4ccb-1ba3085aa7 by eki@lunar.app\nArtifact-created-by: Emil Ingerslev <eki@lunar.app>",
			commitInfo: CommitInfo{
				ArtifactID:  "test-s3-push-f4440b4ccb-1ba3085aa7",
				AuthorEmail: "eki@lunar.app",
				AuthorName:  "Emil Ingerslev",
				Service:     "product",
			},
			err: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ExtractInfoFromCommit(tc.commitMessage)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.commitInfo, info, "commitInfo not as expected")
		})
	}
}

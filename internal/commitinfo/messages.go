package commitinfo

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/intent"
)

// ReleaseCommitMessage returns an artifact release commit message.
func ReleaseCommitMessage(env, service, artifactID string, intent intent.Intent, artifactAuthor, releaseAuthor PersonInfo) string {
	return CommitInfo{
		Environment:       env,
		Service:           service,
		ArtifactID:        artifactID,
		Intent:            intent,
		ArtifactCreatedBy: artifactAuthor,
		ReleasedBy:        releaseAuthor,
	}.String()
}

// PolicyUpdateApplyCommitMessage returns an apply policy commit message.
func PolicyUpdateApplyCommitMessage(env, service, policy string) string {
	return fmt.Sprintf("[%s] policy update: apply %s in '%s'", service, policy, env)
}

// PolicyUpdateDeleteCommitMessage returns a delete policy commit message.
func PolicyUpdateDeleteCommitMessage(service string) string {
	return fmt.Sprintf("[%s] policy update: delete policies", service)
}

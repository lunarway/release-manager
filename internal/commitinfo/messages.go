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

// RollbackCommitMessage returns an artifact rollback commit message.
func RollbackCommitMessage(env, service, oldArtifactID, newArtifactID, authorEmail string) string {
	return fmt.Sprintf("[%s/%s] rollback %s to %s by %s", env, service, oldArtifactID, newArtifactID, authorEmail)
}

// PolicyUpdateApplyCommitMessage returns an apply policy commit message.
func PolicyUpdateApplyCommitMessage(env, service, policy string) string {
	return fmt.Sprintf("[%s] policy update: apply %s in '%s'", service, policy, env)
}

// PolicyUpdateDeleteCommitMessage returns a delete policy commit message.
func PolicyUpdateDeleteCommitMessage(service string) string {
	return fmt.Sprintf("[%s] policy update: delete policies", service)
}

func FullMessage(msg, authorName, authorEmail, committerName, committerEmail string) string {
	return fmt.Sprintf("%s\nArtifact-created-by: %s <%s>\nArtifact-released-by: %s <%s>", msg, authorName, authorEmail, committerName, committerEmail)
}

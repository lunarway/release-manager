package git

import "fmt"

// ArtifactCommitMessage returns an artifact commit message.
func ArtifactCommitMessage(service, artifactID, author string) string {
	return fmt.Sprintf("[%s] artifact %s by %s", service, artifactID, author)
}

// ReleaseCommitMessage returns an artifact release commit message.
func ReleaseCommitMessage(env, service, artifactID, authorEmail string) string {
	return fmt.Sprintf("[%s/%s] release %s by %s", env, service, artifactID, authorEmail)
}

// RollbackCommitMessage returns an artifact rollback commit message.
func RollbackCommitMessage(env, service, oldArtifactID, newArtifactID, authorEmail string) string {
	return fmt.Sprintf("[%s/%s] rollback %s to %s by %s", env, service, oldArtifactID, newArtifactID, authorEmail)
}

// PolicyUpdateApplyCommitMessage returns an apply policy commit message.
func PolicyUpdateApplyCommitMessage(env, service, branch, policy string) string {
	return fmt.Sprintf("[%s] policy update: apply %s from '%s' to '%s'", service, policy, branch, env)
}

// PolicyUpdateDeleteCommitMessage returns a delete policy commit message.
func PolicyUpdateDeleteCommitMessage(service string) string {
	return fmt.Sprintf("[%s] policy update: delete policies", service)
}

package artifact

import "strings"

// NormalizeBranch returns branch with "/" replaced by "_", the form branches
// take in artifact IDs and in Application.Branch of artifacts from Dagger
// plans. Artifacts from older plans store the branch as is, so compare
// branches in this form to match both.
func NormalizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "_")
}

// SameBranch reports whether a and b name the same branch, whether or not
// either of them is normalized.
func SameBranch(a, b string) bool {
	return NormalizeBranch(a) == NormalizeBranch(b)
}

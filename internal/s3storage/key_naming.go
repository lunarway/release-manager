package s3storage

import (
	"fmt"
)

func getObjectKeyName(service string, artifactID string) string {
	return fmt.Sprintf("%s/%s", service, artifactID)
}
func getServiceObjectKeyPrefix(service string) string {
	return fmt.Sprintf("%s/", service)
}
func getServiceAndBranchObjectKeyPrefix(service, branch string) string {
	return fmt.Sprintf("%s/%s-", service, branch)
}
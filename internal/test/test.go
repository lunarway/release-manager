package test

import (
	"os"
	"testing"
)

// RabbitMQIntegration skips the test if no RabbitMQ integration test host name
// is available as an environment variable. If it is available its value is
// returned.
func RabbitMQIntegration(t *testing.T) string {
	host := os.Getenv("RELEASE_MANAGER_INTEGRATION_RABBITMQ_HOST")
	if host == "" {
		t.Skip("RabbitMQ integration tests not enabled as RELEASE_MANAGER_INTEGRATION_RABBITMQ_HOST is empty")
	}
	return host
}

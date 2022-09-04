package github

import (
	"context"
	"os"
	"testing"
)

func TestService_TagRepo(t *testing.T) {
	token := os.Getenv("GITHUB_API_TOKEN")
	if token == "" {
		t.Skip("GITHUB_API_TOKEN not supplied")
	}
	s := Service{
		Token: token,
	}
	err := s.TagRepo(context.Background(), "lunar-way-product-service", "dev", "5c59a4f3bb44a8014dc344440fb29844f91b8c79")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

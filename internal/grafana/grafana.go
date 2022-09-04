package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

type Service struct {
	Logger       *log.Logger
	Environments map[string]Environment
}

type Environment struct {
	APIKey  string
	BaseURL string
}

type AnnotateRequest struct {
	What string   `json:"what,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Data string   `json:"data,omitempty"`
}

type AnnotateResponse struct {
	Message string `json:"message,omitempty"`
	Id      int64  `json:"id,omitempty"`
}

var ErrEnvironmentNotConfigured = errors.New("environment not configured")

func (s *Service) Annotate(ctx context.Context, env string, body AnnotateRequest) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(body)
	if err != nil {
		return err
	}

	e, ok := s.Environments[env]
	if !ok {
		return ErrEnvironmentNotConfigured
	}

	req, err := http.NewRequest(http.MethodPost, e.BaseURL+"/api/annotations/graphite", b)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+e.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	logger := s.Logger.WithContext(ctx)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Infof("grafana: response body: %s", body)
		return errors.New("grafana: status code not ok")
	}
	var responseBody AnnotateResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&responseBody)
	if err != nil {
		return err
	}
	logger.Infof("grafana: AnnotateResponse: message: %s, id: %d", responseBody.Message, responseBody.Id)
	return nil
}

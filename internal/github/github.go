package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

type Service struct {
	Token string
}

func (s *Service) req(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := url.URL{
		Host:   "api.github.com",
		Scheme: "https",
		Path:   path,
	}
	req, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("lunarway", s.Token)
	return req, nil
}

// TagRepo tags repository at a specific git sha. If tag already exists it is updated.
func (s *Service) TagRepo(ctx context.Context, repository, tag, sha string) error {
	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(map[string]string{
		"ref": fmt.Sprintf("refs/tags/%s", tag),
		"sha": sha,
	})
	if err != nil {
		return err
	}
	req, err := s.req(ctx, http.MethodPost, fmt.Sprintf("repos/lunarway/%s/git/refs", repository), b)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	logger := log.WithContext(ctx)
	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusUnprocessableEntity:
		errorBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("internal/github: failed too read error body of request")
		}
		logger.Info("internal/github: failed to create tag: trying to update instead: http status %v: body: %s", resp.Status, errorBody)
		err = s.updateTag(ctx, repository, tag, sha)
		if err != nil {
			return errors.WithMessage(err, "update tag")
		}
		return nil
	default:
		errorBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("internal/github: failed too read error body of request")
		}
		return fmt.Errorf("internal/github: http request failed: %s %s: status %v: body: %s", req.Method, req.URL, resp.Status, errorBody)
	}
}

func (s *Service) updateTag(ctx context.Context, repository, tag, sha string) error {
	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(map[string]interface{}{
		"sha":   sha,
		"force": true,
	})
	if err != nil {
		return err
	}
	req, err := s.req(ctx, http.MethodPatch, fmt.Sprintf("repos/lunarway/%s/git/refs/tags/%s", repository, tag), b)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithContext(ctx).Errorf("internal/github: failed too read error body of request")
		}
		return fmt.Errorf("internal/github: http request failed: %s %s: status %v: body: %s", req.Method, req.URL, resp.Status, errorBody)
	}
	return nil
}

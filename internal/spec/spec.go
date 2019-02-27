package spec

import (
	"encoding/json"
	"os"
	"time"
)

type Spec struct {
	Git        Git        `json:"git,omitempty"`
	CI         CI         `json:"ci,omitempty"`
	Squad      string     `json:"squad,omitempty"`
	Repository Repository `json:"repository,omitempty"`
	Stages     []Stage    `json:"stages,omitempty"`
}

type Git struct {
	SHA       string `json:"sha,omitempty"`
	Author    string `json:"author,omitempty"`
	Committer string `json:"committer,omitempty"`
	Message   string `json:"message,omitempty"`
}

type CI struct {
	JobURL string    `json:"jobUrl,omitempty"`
	Start  time.Time `json:"start,omitempty"`
	End    time.Time `json:"end,omitempty"`
}

type Repository struct {
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type Stage struct {
	ID   string      `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type BuildData struct {
	Image         string `json:"image,omitempty"`
	Tag           string `json:"tag,omitempty"`
	DockerVersion string `json:"docker_version,omitempty"`
}

type PushData struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
	URL   string `json:"url,omitempty"`
}

type TestData struct {
	URL         string     `json:"url,omitempty"`
	TestResults TestResult `json:"test_results,omitempty"`
}

type TestResult struct {
	Passed  int `json:"passed,omitempty"`
	Failed  int `json:"failed,omitempty"`
	Skipped int `json:"skipped,omitempty"`
}

type SnykDockerData struct {
	Tag             string              `json:"tag,omitempty"`
	SnykVersion     string              `json:"snyk_version,omitempty"`
	URL             string              `json:"url,omitempty"`
	BaseImage       string              `json:"base_image,omitempty"`
	Vulnerabilities VulnerabilityResult `json:"vulnerabilities,omitempty"`
}

type SnykCodeData struct {
	Tag             string              `json:"tag,omitempty"`
	SnykVersion     string              `json:"snyk_version,omitempty"`
	URL             string              `json:"url,omitempty"`
	Language        string              `json:"language,omitempty"`
	Vulnerabilities VulnerabilityResult `json:"vulnerabilities,omitempty"`
}

type VulnerabilityResult struct {
	High   int `json:"high,omitempty"`
	Medium int `json:"medium,omitempty"`
	Low    int `json:"low,omitempty"`
}

func Get(path string) (Spec, error) {
	s, err := os.Open(path)
	if err != nil {
		return Spec{}, err
	}
	defer s.Close()
	var fileSpec Spec
	decoder := json.NewDecoder(s)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&fileSpec)
	if err != nil {
		return Spec{}, err
	}
	return fileSpec, nil
}

func Persist(path string, spec Spec) error {
	s, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer s.Close()
	encode := json.NewEncoder(s)
	err = encode.Encode(spec)
	if err != nil {
		return err
	}
	return nil
}

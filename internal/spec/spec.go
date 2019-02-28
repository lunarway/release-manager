package spec

import (
	"encoding/json"
	"os"
	"time"
)

type Spec struct {
	Application Repository `json:"application,omitempty"`
	CI          CI         `json:"ci,omitempty"`
	Squad       string     `json:"squad,omitempty"`
	Shuttle     Shuttle    `json:"shuttle,omitempty"`
	Stages      []Stage    `json:"stages,omitempty"`
}

type Repository struct {
	SHA       string `json:"sha,omitempty"`
	Author    string `json:"author,omitempty"`
	Committer string `json:"committer,omitempty"`
	Message   string `json:"message,omitempty"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url,omitempty"`
	Provider  string `json:"provider,omitempty"`
}

type Shuttle struct {
	Plan           Repository `json:"plan,omitempty"`
	ShuttleVersion string     `json:"shuttleVersion,omitempty"`
}

type CI struct {
	JobURL string    `json:"jobUrl,omitempty"`
	Start  time.Time `json:"start,omitempty"`
	End    time.Time `json:"end,omitempty"`
}

type Stage struct {
	ID   string      `json:"id,omitempty"`
	Name string      `json:"name,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type BuildData struct {
	Image         string `json:"image,omitempty"`
	Tag           string `json:"tag,omitempty"`
	DockerVersion string `json:"dockerVersion,omitempty"`
}

type PushData struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
	URL   string `json:"url,omitempty"`
}

type TestData struct {
	URL     string     `json:"url,omitempty"`
	Results TestResult `json:"results,omitempty"`
}

type TestResult struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type SnykDockerData struct {
	Tag             string              `json:"tag,omitempty"`
	SnykVersion     string              `json:"snykVersion,omitempty"`
	URL             string              `json:"url,omitempty"`
	BaseImage       string              `json:"baseImage,omitempty"`
	Vulnerabilities VulnerabilityResult `json:"vulnerabilities,omitempty"`
}

type SnykCodeData struct {
	SnykVersion     string              `json:"snykVersion,omitempty"`
	URL             string              `json:"url,omitempty"`
	Language        string              `json:"language,omitempty"`
	Vulnerabilities VulnerabilityResult `json:"vulnerabilities,omitempty"`
}

type VulnerabilityResult struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
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

func Update(path string, f func(Spec) Spec) error {
	s, err := Get(path)
	if err != nil {
		return err
	}

	s = f(s)

	// Persist back to the file
	err = Persist(path, s)
	if err != nil {
		return err
	}
	return nil
}

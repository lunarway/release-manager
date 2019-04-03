package artifact

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrFileNotFound indicates that an artifact was not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrNotParsable indicates that an artifact could not be parsed against the
	// artifact specification.
	ErrNotParsable = errors.New("artifact not parsable")
	// ErrUnknownFields indicates that an artifact contains an unknown field.
	ErrUnknownFields = errors.New("artifact contains unknown fields")
)

type Spec struct {
	ID          string     `json:"id,omitempty"`
	Application Repository `json:"application,omitempty"`
	CI          CI         `json:"ci,omitempty"`
	Squad       string     `json:"squad,omitempty"`
	Shuttle     Shuttle    `json:"shuttle,omitempty"`
	Stages      []Stage    `json:"stages,omitempty"`
}

type Repository struct {
	SHA            string `json:"sha,omitempty"`
	AuthorName     string `json:"authorName,omitempty"`
	AuthorEmail    string `json:"authorEmail,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
	Message        string `json:"message,omitempty"`
	Name           string `json:"name,omitempty"`
	URL            string `json:"url,omitempty"`
	Provider       string `json:"provider,omitempty"`
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
	Image         string `json:"image,omitempty"`
	Tag           string `json:"tag,omitempty"`
	DockerVersion string `json:"dockerVersion,omitempty"`
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

// GetStage returns a stage of the spec with provided stage ID.
func (s *Spec) GetStage(stageID string) (Stage, bool) {
	for _, stage := range s.Stages {
		if stage.ID == stageID {
			return stage, true
		}
	}
	return Stage{}, false
}

func Get(path string) (Spec, error) {
	s, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Spec{}, ErrFileNotFound
		}
		return Spec{}, err
	}
	defer s.Close()
	var fileSpec Spec
	decoder := json.NewDecoder(s)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&fileSpec)
	if err != nil {
		_, ok := err.(*json.SyntaxError)
		if ok {
			return Spec{}, ErrNotParsable
		}
		// there is no other way to detect this error type unfortunately
		// https://github.com/golang/go/blob/277609f844ed9254d25e975f7cf202d042beecc6/src/encoding/json/decode.go#L739
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			return Spec{}, errors.WithMessagef(ErrUnknownFields, "%v", err)
		}
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
	encode.SetIndent("", "  ")
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

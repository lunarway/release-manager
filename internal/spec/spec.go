package spec

import "time"

type Spec struct {
	Git        Git
	Ci         Ci
	Squad      string
	Repository Repository
	Stages     []Stage
}

type Git struct {
	SHA       string `json:"sha,omitempty"`
	Author    string `json:"author,omitempty"`
	Committer string `json:"committer,omitempty"`
	Message   string `json:"message,omitempty"`
}

type Ci struct {
	URL      string
	Duration time.Duration
}

type Repository struct {
	Name     string
	URL      string
	Provider string
}

type Stage struct {
	ID   string
	Name string
	Data interface{}
}

type BuildData struct {
	Image         string
	Tag           string
	DockerVersion string
}

type PushData struct {
	Image string
	Tag   string
	URL   string
}

type TestData struct {
	URL         string
	TestResults TestResult
}

type TestResult struct {
	Passed  int
	Failed  int
	Skipped int
}

type SnykDockerData struct {
	Tag             string
	SnykVersion     string
	URL             string
	BaseImage       string
	Vulnerabilities VulnerabilityResult
}

type SnykCodeData struct {
	Tag             string
	SnykVersion     string
	URL             string
	Language        string
	Vulnerabilities VulnerabilityResult
}

type VulnerabilityResult struct {
	High   int
	Medium int
	Low    int
}

func Get(path string) (Spec, error) {
	return Spec{}, nil
}

package flow

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/copy"
	internalgit "github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
)

var (
	ErrUnknownEnvironment            = errors.New("unknown environment")
	ErrNamespaceNotAllowedByArtifact = errors.New("namespace not allowed by artifact")
	ErrUnknownConfiguration          = errors.New("unknown configuration")
	ErrNothingToRelease              = errors.New("nothing to release")
	ErrReleaseProhibited             = errors.New("release prohibited")
)

type Service struct {
	ArtifactFileName string
	UserMappings     map[string]string
	Slack            *slack.Client
	Git              GitService
	Tracer           tracing.Tracer
	CanRelease       func(ctx context.Context, svc, branch, env string) (bool, error)
	Storage          ArtifactReadStorage
	Policy           *policy.Service

	PublishReleaseArtifactID func(context.Context, ReleaseArtifactIDEvent) error
	PublishNewArtifact       func(context.Context, NewArtifactEvent) error

	MaxRetries int

	// NotifyReleaseHook is triggered in a Go routine when a release is completed.
	// The context.Context is cancelled if the originating flow call is cancelled.
	NotifyReleaseHook func(ctx context.Context, options NotifyReleaseOptions)
}

type NotifyReleaseOptions struct {
	Environment string
	Namespace   string
	Service     string
	Releaser    string
	Spec        artifact.Spec
}

type GitService interface {
	SyncMaster(context.Context) error
	Clone(context.Context, string) (*git.Repository, error)
	MasterPath() string
	Commit(ctx context.Context, rootPath, changesPath, msg string) error
	LocateServiceReleaseRollbackSkip(ctx context.Context, r *git.Repository, env, service string, n uint) (plumbing.Hash, error)
	Checkout(ctx context.Context, rootPath string, hash plumbing.Hash) error
}

// retry tries the function f until max attempts is reached
// If f returns a true bool or a nil error retries are stopped and the error is
// returned.
func (s *Service) retry(ctx context.Context, f func(context.Context, int) (bool, error)) error {
	return try.Do(ctx, s.Tracer, s.MaxRetries, func(ctx context.Context, attempt int) (bool, error) {
		stop, err := f(ctx, attempt)
		if err != nil {
			if errors.Cause(err) == internalgit.ErrBranchBehindOrigin {
				log.WithContext(ctx).Infof("flow/retry: master repo not aligned with origin. Syncing and retrying")
				err := s.Git.SyncMaster(ctx)
				if err != nil {
					return false, errors.WithMessage(err, "sync master")
				}
			}
		}
		return stop, err
	})
}

type Environment struct {
	Tag                   string    `json:"tag,omitempty"`
	Committer             string    `json:"committer,omitempty"`
	Author                string    `json:"author,omitempty"`
	Message               string    `json:"message,omitempty"`
	Date                  time.Time `json:"date,omitempty"`
	BuildURL              string    `json:"buildUrl,omitempty"`
	HighVulnerabilities   int64     `json:"highVulnerabilities,omitempty"`
	MediumVulnerabilities int64     `json:"mediumVulnerabilities,omitempty"`
	LowVulnerabilities    int64     `json:"lowVulnerabilities,omitempty"`
}

type StatusResponse struct {
	DefaultNamespaces bool        `json:"defaultNamespaces,omitempty"`
	Dev               Environment `json:"dev,omitempty"`
	Staging           Environment `json:"staging,omitempty"`
	Prod              Environment `json:"prod,omitempty"`
}

type Actor struct {
	Email string
	Name  string
}

func (s *Service) Status(ctx context.Context, namespace, service string) (StatusResponse, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.Status")
	defer span.Finish()

	defaultNamespaces := namespace == ""
	defaultNamespace := func(env string) string {
		if defaultNamespaces {
			return env
		}
		return namespace
	}
	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "dev")
	devSpec, err := s.releaseSpecification(ctx, releaseLocation{
		Environment: "dev",
		Service:     service,
		Namespace:   defaultNamespace("dev"),
	})
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env dev")
		}
	}
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "staging")
	stagingSpec, err := s.releaseSpecification(ctx, releaseLocation{
		Environment: "staging",
		Service:     service,
		Namespace:   defaultNamespace("staging"),
	})
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env staging")
		}
	}
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "prod")
	prodSpec, err := s.releaseSpecification(ctx, releaseLocation{
		Environment: "prod",
		Service:     service,
		Namespace:   defaultNamespace("prod"),
	})
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env prod")
		}
	}
	defer span.Finish()

	return StatusResponse{
		DefaultNamespaces: defaultNamespaces,
		Dev:               mapSpec(devSpec),
		Staging:           mapSpec(stagingSpec),
		Prod:              mapSpec(prodSpec),
	}, nil
}

func mapSpec(spec artifact.Spec) Environment {
	return Environment{
		Tag:                   spec.ID,
		Committer:             spec.Application.CommitterName,
		Author:                spec.Application.AuthorName,
		Message:               spec.Application.Message,
		Date:                  spec.CI.End,
		BuildURL:              spec.CI.JobURL,
		HighVulnerabilities:   calculateHighTotalVulnerabilties(spec),
		MediumVulnerabilities: calculateMediumTotalVulnerabilties(spec),
		LowVulnerabilities:    calculateLowTotalVulnerabilties(spec),
	}
}

func calculateHighTotalVulnerabilties(s artifact.Spec) int64 {
	return calculateTotalVulnerabilties(s, func(v artifact.VulnerabilityResult) int {
		return v.High
	})
}

func calculateMediumTotalVulnerabilties(s artifact.Spec) int64 {
	return calculateTotalVulnerabilties(s, func(v artifact.VulnerabilityResult) int {
		return v.Medium
	})
}

func calculateLowTotalVulnerabilties(s artifact.Spec) int64 {
	return calculateTotalVulnerabilties(s, func(v artifact.VulnerabilityResult) int {
		return v.Low
	})
}

func calculateTotalVulnerabilties(s artifact.Spec, field func(artifact.VulnerabilityResult) int) int64 {
	result := float64(0)
	for _, stage := range s.Stages {
		var vulnerabilities artifact.VulnerabilityResult
		if stage.ID == artifact.StageIDSnykCode {
			data := stage.Data.(artifact.SnykCodeData)
			vulnerabilities = data.Vulnerabilities
		}
		if stage.ID == artifact.StageIDSnykDocker {
			data := stage.Data.(artifact.SnykDockerData)
			vulnerabilities = data.Vulnerabilities
		}
		result += float64(field(vulnerabilities))
	}
	return int64(result + 0.5)
}

type releaseLocation struct {
	Environment string
	Namespace   string
	Service     string
}

func (s *Service) releaseSpecification(ctx context.Context, location releaseLocation) (artifact.Spec, error) {
	releasePath, err := releasePath(s.Git.MasterPath(), location.Service, location.Environment, location.Namespace)
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "get release path")
	}
	artifactPath := path.Join(releasePath, s.ArtifactFileName)
	spec, err := artifact.Get(artifactPath)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "artifact path '%s'", artifactPath)
	}
	return spec, nil
}

func envSpec(root, artifactFileName, service, env, namespace string) (artifact.Spec, error) {
	releasePath, err := releasePath(root, service, env, namespace)
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "get release path")
	}
	return artifact.Get(path.Join(releasePath, artifactFileName))
}

// releasePath returns the path of a specific release.
func releasePath(root, service, env, namespace string) (string, error) {
	releasePath := root // start with root
	pathsToJoin := []string{
		env,
		"releases",
		namespace,
		service,
	}
	var err error
	for _, p := range pathsToJoin {
		releasePath, err = securejoin.SecureJoin(releasePath, p)
		if err != nil {
			return "", errors.WithMessagef(err, "join '%s' to path", p)
		}
	}
	return releasePath, nil
}

// PushArtifactToReleaseManager pushes an artifact to the release manager
func PushArtifactToReleaseManager(ctx context.Context, releaseManagerClient *httpinternal.Client, artifactFileName, resourceRoot string) (string, error) {
	artifactSpecPath := path.Join(resourceRoot, artifactFileName)
	artifactSpec, err := artifact.Get(artifactSpecPath)
	if err != nil {
		return "", errors.WithMessagef(err, "path '%s'", artifactSpecPath)
	}

	files := listFiles(resourceRoot)

	zipContent, err := zipFiles(files)
	if err != nil {
		return "", errors.WithMessage(err, "zip artifact failed")
	}
	log.WithFields("artifactID", artifactSpec.ID, "artifactFiles", files).Infof("artifact zip created for %s", artifactSpec.ID)

	zipMD5 := md5.New()
	_, err = zipMD5.Write(zipContent)
	if err != nil {
		return "", errors.Wrap(err, "calculate md5 hash")
	}
	zipMD5s := base64.StdEncoding.EncodeToString(zipMD5.Sum(nil))

	log.WithFields("artifactID", artifactSpec.ID, "artifactFiles", files).Infof("calculated zip md5: %x", zipMD5s)

	path, err := releaseManagerClient.URL(fmt.Sprintf("artifacts/create"))
	if err != nil {
		return "", errors.WithMessage(err, "push artifact URL generation failed")
	}

	resp := httpinternal.ArtifactUploadResponse{}
	err = releaseManagerClient.Do(http.MethodPost, path, httpinternal.ArtifactUploadRequest{
		Artifact: artifactSpec,
		MD5:      zipMD5s,
	}, &resp)
	if err != nil {
		return "", errors.WithMessage(err, "create artifact request failed")
	}
	log.WithFields("artifactID", artifactSpec.ID, "uploadURL", resp.ArtifactUploadURL).Infof("artifact upload URL created for %s", artifactSpec.ID)

	err = uploadFile(resp.ArtifactUploadURL, zipContent, string(zipMD5s))
	if err != nil {
		return "", errors.WithMessage(err, "upload artifact failed")
	}

	log.WithFields("artifactID", artifactSpec.ID).Infof("uploaded artifact %s", artifactSpec.ID)

	return artifactSpec.ID, nil
}

func listFiles(path string) []fileInfo {
	var files []fileInfo
	fmt.Printf("Files in path '%s'\n", path)
	err := filepath.Walk(path,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("failed to walk dir: %v\n", err)
				return nil
			}
			if !info.IsDir() {
				fullPath, err := filepath.Abs(filePath)
				if err != nil {
					fmt.Printf("failed to generate absolute path for %s: %v\n", filePath, err)
					return nil
				}
				relativePath, err := filepath.Rel(path, filePath)
				if err != nil {
					fmt.Printf("failed to generate relative path for %s: %v\n", filePath, err)
					return nil
				}
				files = append(files, fileInfo{
					fullPath:     fullPath,
					relativePath: relativePath,
				})
			}
			fmt.Printf("  %s\n", filePath)
			return nil
		})
	if err != nil {
		fmt.Printf("failed to read dir: %v\n", err)
	}
	return files
}

type fileInfo struct {
	fullPath     string
	relativePath string
}

func (s *Service) notifyRelease(ctx context.Context, opts NotifyReleaseOptions) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.notifyRelease")
	defer span.Finish()
	if s.NotifyReleaseHook != nil {
		go s.NotifyReleaseHook(noCancel{ctx: ctx}, opts)
	}
}

// noCancel is a context.Context that does not propagate cancellations.
type noCancel struct {
	ctx context.Context
}

func (c noCancel) Deadline() (time.Time, bool)       { return time.Time{}, false }
func (c noCancel) Done() <-chan struct{}             { return nil }
func (c noCancel) Err() error                        { return nil }
func (c noCancel) Value(key interface{}) interface{} { return c.ctx.Value(key) }

func (s *Service) cleanCopy(ctx context.Context, src, dest string) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.cleanCopy")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "remove destination")
	err := os.RemoveAll(dest)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "remove destination path")
	}
	span, _ = s.Tracer.FromCtx(ctx, "create destination dir")
	err = os.MkdirAll(dest, os.ModePerm)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "create destination dir")
	}
	span, _ = s.Tracer.FromCtx(ctx, "copy files")
	span.Finish()
	err = copy.CopyDir(ctx, src, dest)
	if err != nil {
		if errors.Cause(err) == copy.ErrUnknownSource {
			return ErrUnknownConfiguration
		}
		return errors.WithMessage(err, "copy files")
	}
	return nil
}

func zipFiles(files []fileInfo) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, file := range files {
		f, err := w.Create(file.relativePath)
		if err != nil {
			return nil, err
		}
		fileContent, err := ioutil.ReadFile(file.fullPath)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(fileContent)
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func uploadFile(url string, fileContent []byte, md5 string) error {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(fileContent))
	if err != nil {
		return err
	}

	req.Header.Set("Content-MD5", md5)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed upload file to %s with status code %v and request id %v and and also got an error reading body %w", url, resp.StatusCode, resp.Header["X-Amz-Request-Id"], err)
		}
		return fmt.Errorf("failed upload file to %s with status code %v and request id %v and body %s", url, resp.StatusCode, resp.Header["X-Amz-Request-Id"], string(body))
	}

	return nil
}

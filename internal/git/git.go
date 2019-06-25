package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

var (
	ErrNothingToCommit  = errors.New("nothing to commit")
	ErrReleaseNotFound  = errors.New("release not found")
	ErrArtifactNotFound = errors.New("artifact not found")
)

type Service struct {
	Tracer            opentracing.Tracer
	SSHPrivateKeyPath string
	ConfigRepoURL     string

	masterMutex sync.Mutex
	masterPath  string
	master      *git.Repository
}

func (s *Service) span(ctx context.Context, op string) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContextWithTracer(ctx, s.Tracer, op)
}

// InitMasterRepo clones the configuration repository into a master directory.
func (s *Service) InitMasterRepo(ctx context.Context) (func(context.Context), error) {
	span, ctx := s.span(ctx, "git.InitMasterRepo")
	defer span.Finish()
	path, close, err := TempDir(ctx, s.Tracer, "k8s-master-clone")
	if err != nil {
		close(ctx)
		return nil, errors.WithMessage(err, "get temporary directory")
	}
	repo, err := s.clone(ctx, path)
	if err != nil {
		close(ctx)
		return nil, errors.WithMessagef(err, "clone into '%s'", path)
	}
	s.masterMutex.Lock()
	defer s.masterMutex.Unlock()
	s.master = repo
	s.masterPath = path
	log.Infof("Master repo cloned into '%s'", path)
	return close, nil
}

func (s *Service) clone(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.span(ctx, "git.clone")
	defer span.Finish()
	authSSH, err := ssh.NewPublicKeysFromFile("git", s.SSHPrivateKeyPath, "")
	if err != nil {
		return nil, errors.WithMessage(err, "public keys from file")
	}
	err = os.RemoveAll(destination)
	if err != nil {
		return nil, errors.WithMessage(err, "remove existing destination")
	}

	r, err := git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:  s.ConfigRepoURL,
		Auth: authSSH,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "clone repo")
	}
	return r, nil
}

// SyncMaster pulls latest changes from master repo.
func (s *Service) SyncMaster(ctx context.Context) error {
	span, ctx := s.span(ctx, "git.SyncMaster")
	defer span.Finish()
	authSSH, err := ssh.NewPublicKeysFromFile("git", s.SSHPrivateKeyPath, "")
	if err != nil {
		return errors.WithMessage(err, "public keys from file")
	}
	span, _ = s.span(ctx, "lock mutex")
	s.masterMutex.Lock()
	defer s.masterMutex.Unlock()
	span.Finish()

	span, _ = s.span(ctx, "fetch")
	err = s.master.FetchContext(ctx, &git.FetchOptions{
		Auth: authSSH,
	})
	span.Finish()
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return errors.WithMessage(err, "fetch changes")
	}
	w, err := s.master.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	span, _ = s.span(ctx, "pull")
	err = w.PullContext(ctx, &git.PullOptions{
		Auth: authSSH,
	})
	defer span.Finish()
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return errors.WithMessage(err, "pull latest")
	}
	return nil
}

// Clone returns a Git repository copy from the master repository.
func (s *Service) Clone(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.span(ctx, "git.Clone")
	defer span.Finish()
	return s.copyMaster(ctx, destination)
}

func (s *Service) copyMaster(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.span(ctx, "git.copyMaster")
	defer span.Finish()
	span, _ = s.span(ctx, "remove destination")
	err := os.RemoveAll(destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "remove existing destination")
	}
	span, _ = s.span(ctx, "lock mutex")
	s.masterMutex.Lock()
	defer s.masterMutex.Unlock()
	span.Finish()
	span, _ = s.span(ctx, "copy to destination")
	err = copy.Copy(s.masterPath, destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessagef(err, "copy master from '%s'", s.masterPath)
	}
	span, _ = s.span(ctx, "open repo")
	r, err := git.PlainOpen(destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "open repo")
	}
	return r, nil
}

func (s *Service) Checkout(ctx context.Context, r *git.Repository, hash plumbing.Hash) error {
	span, ctx := s.span(ctx, "git.Checkout")
	defer span.Finish()
	workTree, err := r.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err != nil {
		return errors.WithMessage(err, "checkout hash")
	}
	return nil
}

// LocateRelease traverses the git log to find a release commit with id
// artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateRelease(ctx context.Context, r *git.Repository, artifactID string) (plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateRelease")
	defer span.Finish()
	return locate(r, locateReleaseCondition(artifactID), ErrReleaseNotFound)
}

func locateReleaseCondition(artifactID string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)release %s$`, regexp.QuoteMeta(artifactID)))
	return func(commitMsg string) bool {
		if artifactID == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateServiceRelease traverses the git log to find a release
// commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateServiceRelease(ctx context.Context, r *git.Repository, env, service string) (plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateServiceRelease")
	defer span.Finish()
	return locate(r, locateServiceReleaseCondition(env, service), ErrReleaseNotFound)
}

func locateServiceReleaseCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] release`, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateEnvRelease traverses the git log to find a release
// commit for a specified environment and artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateEnvRelease(ctx context.Context, r *git.Repository, env, artifactID string) (plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateEnvRelease")
	defer span.Finish()
	return locate(r, locateEnvReleaseCondition(env, artifactID), ErrReleaseNotFound)
}

func locateEnvReleaseCondition(env, artifactId string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/.*] release %s$`, regexp.QuoteMeta(env), regexp.QuoteMeta(artifactId)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if artifactId == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateServiceReleaseRollbackSkip traverses the git log to find a release or
// rollback commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage or RollbackCommitMessage.
func (s *Service) LocateServiceReleaseRollbackSkip(ctx context.Context, r *git.Repository, env, service string, n uint) (plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateServiceReleaseRollbackSkip")
	defer span.Finish()
	return locate(r, locateServiceReleaseRollbackSkipCondition(env, service, n), ErrReleaseNotFound)
}

func locateServiceReleaseRollbackSkipCondition(env, service string, n uint) conditionFunc {
	return func(commitMsg string) bool {
		releaseOK := locateServiceReleaseCondition(env, service)(commitMsg)
		rollbackOK := locateServiceRollbackCondition(env, service)(commitMsg)
		ok := releaseOK || rollbackOK
		if !ok {
			return false
		}
		if n == 0 {
			return true
		}
		n--
		return false
	}
}

func locateServiceRollbackCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] rollback `, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateArtifact traverses the git log to find an artifact commit with id
// artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ArtifactCommitMessage.
func (s *Service) LocateArtifact(ctx context.Context, r *git.Repository, artifactID string) (plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateArtifact")
	defer span.Finish()
	return locate(r, locateArtifactCondition(artifactID), ErrArtifactNotFound)
}

func locateArtifactCondition(artifactID string) conditionFunc {
	artifactRegex := regexp.MustCompile(fmt.Sprintf(`(?i)artifact %s `, regexp.QuoteMeta(artifactID)))
	return func(commitMsg string) bool {
		if artifactID == "" {
			return false
		}
		return artifactRegex.MatchString(commitMsg)
	}
}

// LocateArtifacts traverses the git log to find artifact commits for a service.
//
// It expects the commit to have a commit messages as the one returned by
// ArtifactCommitMessage.
func (s *Service) LocateArtifacts(ctx context.Context, r *git.Repository, service string, n int) ([]plumbing.Hash, error) {
	span, _ := s.span(ctx, "git.LocateArtifacts")
	defer span.Finish()
	return locateN(r, locateArtifactServiceCondition(service), ErrArtifactNotFound, n)
}

func locateArtifactServiceCondition(service string) conditionFunc {
	artifactRegex := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s] artifact `, regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if service == "" {
			return false
		}
		return artifactRegex.MatchString(commitMsg)
	}
}

type conditionFunc func(commitMsg string) bool

func locate(r *git.Repository, condition conditionFunc, notFoundErr error) (plumbing.Hash, error) {
	hashes, err := locateN(r, condition, notFoundErr, 1)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	// locateN will return an error when reaching the end of the git log.
	// So if there is no error we must have found at least one match.
	return hashes[0], nil
}

func locateN(r *git.Repository, condition conditionFunc, notFoundErr error, n int) ([]plumbing.Hash, error) {
	var hashes []plumbing.Hash
	ref, err := r.Head()
	if err != nil {
		return nil, errors.WithMessage(err, "retrieve HEAD branch")
	}
	cIter, err := r.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, errors.WithMessage(err, "retrieve commit history")
	}
	for {
		commit, err := cIter.Next()
		if err != nil {
			if err == io.EOF {
				return hashes, notFoundErr
			}
			return nil, errors.WithMessage(err, "retrieve commit")
		}
		if condition(commit.Message) {
			hashes = append(hashes, commit.Hash)
		}
		if len(hashes) >= n {
			return hashes, nil
		}
	}
}

func (s *Service) Commit(ctx context.Context, repo *git.Repository, changesPath, authorName, authorEmail, committerName, committerEmail, msg string) error {
	span, ctx := s.span(ctx, "git.Commit")
	defer span.Finish()
	w, err := repo.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	err = w.AddGlob(changesPath)
	if err != nil {
		return errors.WithMessage(err, "add changes")
	}

	status, err := w.Status()
	if err != nil {
		return errors.WithMessage(err, "status")
	}
	log.Infof("internal/git: Commit status:\n%s", status)
	// if commit is empty
	if status.IsClean() {
		log.Debugf("internal/git: Commit: message '%s': nothing to commit", msg)
		return ErrNothingToCommit
	}

	span, _ = s.span(ctx, "commit")
	_, err = w.Commit(msg, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  committerName,
			Email: committerEmail,
			When:  time.Now(),
		},
	})
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "commit")
	}

	authSSH, err := ssh.NewPublicKeysFromFile("git", s.SSHPrivateKeyPath, "")
	if err != nil {
		return errors.WithMessage(err, "public keys from file")
	}

	// TODO: this could be made optional if needed
	span, _ = s.span(ctx, "push")
	defer span.Finish()
	err = repo.PushContext(ctx, &git.PushOptions{Auth: authSSH})
	if err != nil {
		return errors.WithMessage(err, "push")
	}
	return nil
}

// userHomeDir returns the home directory of the current user.
//
// It handles windows, linux and darwin operating systems by inspecting
// runtime.GOOS.
func userHomeDir() string {
	switch runtime.GOOS {
	case "windows":
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	case "linux":
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
		fallthrough
	default:
		return os.Getenv("HOME")
	}
}

// CommitterDetails returns name and email read for a Git configuration file.
//
// Configuration files are read first in the local git repository (if available)
// and then read the global Git configuration.
func CommitterDetails() (string, string, error) {
	var paths []string
	pwd, err := os.Getwd()
	if err == nil {
		paths = append(paths, path.Join(pwd, ".git", "config"))
	}
	paths = append(paths, path.Join(userHomeDir(), ".gitconfig"))
	return credentials(paths...)
}

// credentials will try to read user name and email from provided paths.
func credentials(paths ...string) (string, string, error) {
	for _, path := range paths {
		c, err := parseConfig(path)
		if err != nil {
			continue
		}
		committerName := c.Section("user").Option("name")
		committerEmail := c.Section("user").Option("email")
		if committerEmail == "" {
			continue
		}
		if committerName == "" {
			continue
		}
		return committerName, committerEmail, nil
	}
	return "", "", errors.Errorf("failed to read Git credentials from paths: %v", paths)
}

// parseConfig returns the Git configuration parsed from provided path.
func parseConfig(path string) (config.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return config.Config{}, err
	}
	decoder := config.NewDecoder(file)
	var c config.Config
	err = decoder.Decode(&c)
	if err != nil {
		return config.Config{}, err
	}
	return c, nil
}

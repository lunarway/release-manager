package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/cmd/hamctl/command/completion"
	"github.com/lunarway/release-manager/internal/artifact"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type LoggerFunc = func(string, ...any)

type ReleaseArtifactMultipleEnvironments interface {
	ReleaseArtifactIDMultipleEnvironments(
		service string, environments []string, artifactID string, releaseIntent intent.Intent,
	) ([]actions.ReleaseResult, error)
}

type branchGetter func() string

// GitRepository is the Git repository hamctl is run in.
type GitRepository interface {
	HeadSHA(ctx context.Context) (string, error)
	CommitExists(ctx context.Context, sha string) bool
	FetchBranch(ctx context.Context, branch string) error
	IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error)
	IsPushed(ctx context.Context, sha, branch string) (bool, error)
	CountCommits(ctx context.Context, from, to string) (int, error)
}

const (
	// releaseWaitPollInterval is how often the latest artifact is looked up
	// while waiting for the artifact of HEAD.
	releaseWaitPollInterval = 10 * time.Second
	// recentArtifactsCount is the number of the newest artifacts of a branch
	// searched for the artifact of HEAD before waiting for it.
	recentArtifactsCount = 20
)

const releaseExample = `Release artifact 'master-482c9d808e-3bf40478e5' from service 'product' into environment 'dev':

  hamctl release --service product --env dev --artifact master-482c9d808e-3bf40478e5

Release latest artifact from branch 'master' of service 'product' into environment 'dev':

  hamctl release --service product --env dev --branch master

Release latest artifact from current branch of service 'product' into environment 'dev':

  hamctl release --service product --env dev --current-branch

Release artifact built from HEAD of current branch of service 'product' into environment 'dev',
waiting for CI to build it:

  hamctl release --service product --env dev --current-branch --wait`

func NewRelease(
	client *httpinternal.Client,
	service *string,
	logger LoggerFunc,
	releaseClient ReleaseArtifactMultipleEnvironments,
	branchGetter branchGetter,
	repository GitRepository,
	pollInterval time.Duration,
) *cobra.Command {
	var flags releaseFlags
	release := func(artifactID string, releaseIntent intent.Intent) error {
		resps, err := releaseClient.ReleaseArtifactIDMultipleEnvironments(
			*service, flags.environments, artifactID, releaseIntent)
		if err != nil {
			return err
		}
		for _, resp := range resps {
			printReleaseResponse(logger, resp)
		}
		return nil
	}
	releaseBranch := func(branch, artifactID string) error {
		logger("Release of service %s using branch %s\n", *service, branch)
		return release(artifactID, intent.NewReleaseBranch(branch))
	}
	var command = &cobra.Command{
		Use:     "release",
		Short:   `Release a specific artifact or latest artifact from a branch into a specific environment.`,
		Example: releaseExample,
		Args:    cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, _ []string) error {
			flags.environments = trimEmptyValues(flags.environments)
			err := flags.validate()
			if err != nil {
				return err
			}
			switch {
			case flags.currentBranch:
				branch := branchGetter()
				if branch == "" {
					return errors.New("Could not determine current branch. Specify --branch or --artifact explicitly")
				}
				finder := newHeadArtifactFinder(client, repository, logger, *service, branch)
				var spec artifact.Spec
				if flags.wait {
					spec, err = finder.wait(c.Context(), flags.waitTimeout, pollInterval)
				} else {
					spec, err = finder.latest(c.Context())
				}
				if err != nil {
					return err
				}
				return releaseBranch(branch, spec.ID)

			case flags.branch != "":
				spec, err := actions.LatestArtifactFromBranch(client, *service, flags.branch)
				if err != nil {
					return err
				}
				return releaseBranch(flags.branch, spec.ID)

			default:
				logger("Release of service: %s\n", *service)
				return release(flags.artifactID, intent.NewReleaseArtifact())
			}
		},
	}
	flags.register(command)
	return command
}

// releaseFlags are the flags of the release command.
type releaseFlags struct {
	environments  []string
	branch        string
	artifactID    string
	currentBranch bool
	wait          bool
	waitTimeout   time.Duration
}

func (f *releaseFlags) register(command *cobra.Command) {
	command.Flags().StringSliceVarP(&f.environments, "env", "e", nil,
		"Comma separated list of environments to release to (required)")
	// errors are skipped here as the only case they can occur are if the flag
	// does not exist on the command.
	//nolint:errcheck,gosec
	command.MarkFlagRequired("env")
	completion.FlagAnnotation(command, "env", "__hamctl_get_environments")
	command.Flags().StringVarP(&f.branch, "branch", "b", "",
		"release latest artifact from this branch (mutually exclusive with --artifact and --current-branch)")
	completion.FlagAnnotation(command, "branch", "__hamctl_get_branches")
	command.Flags().StringVar(&f.artifactID, "artifact", "",
		"release this artifact id (mutually exclusive with --branch and --current-branch)")
	command.Flags().BoolVarP(&f.currentBranch, "current-branch", "c", false,
		"release latest artifact from the current branch (mutually exclusive with --artifact and --branch)")
	command.Flags().BoolVar(&f.wait, "wait", false,
		"release the artifact built from HEAD, waiting for it if needed, or a newer artifact containing HEAD "+
			"(requires --current-branch and HEAD pushed to origin)")
	command.Flags().DurationVar(&f.waitTimeout, "wait-timeout", 30*time.Minute,
		"how long --wait waits for the artifact of HEAD")
}

// validate returns an error if the flags do not describe a single release.
func (f *releaseFlags) validate() error {
	switch {
	case len(f.environments) == 0:
		return errors.New("--env must contain at least one value")
	case f.wait && !f.currentBranch:
		return errors.New("--wait requires --current-branch")
	case f.branch != "" && f.currentBranch:
		return errors.New("--branch and --current-branch cannot both be specified")
	case f.branch != "" && f.artifactID != "":
		return errors.New("--branch and --artifact cannot both be specified")
	case f.currentBranch && f.artifactID != "":
		return errors.New("--current-branch and --artifact cannot both be specified")
	case f.branch == "" && f.artifactID == "" && !f.currentBranch:
		return errors.New("--branch, --current-branch or --artifact is required")
	default:
		return nil
	}
}

func trimEmptyValues(values []string) []string {
	var trimmed []string
	for _, v := range values {
		t := strings.TrimSpace(v)
		if t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return trimmed
}

// commitRelation is how a commit relates to HEAD.
type commitRelation int

const (
	// commitCurrent is HEAD.
	commitCurrent commitRelation = iota
	// commitOlder is an ancestor of HEAD.
	commitOlder
	// commitNewer is a descendant of HEAD.
	commitNewer
	// commitDiverged is neither HEAD, an ancestor nor a descendant of HEAD, or
	// is unknown to the repository even after fetching the branch.
	commitDiverged
)

// headArtifactFinder finds the artifacts of a branch to release for the commit
// HEAD points to in the local repository.
type headArtifactFinder struct {
	client     *httpinternal.Client
	repository GitRepository
	logger     LoggerFunc
	service    string
	branch     string
	// fetched holds the commits the branch has been fetched from origin to
	// look for, so that it is fetched at most once per commit.
	fetched map[string]bool
}

func newHeadArtifactFinder(
	client *httpinternal.Client,
	repository GitRepository,
	logger LoggerFunc,
	service, branch string,
) *headArtifactFinder {
	return &headArtifactFinder{
		client:     client,
		repository: repository,
		logger:     logger,
		service:    service,
		branch:     branch,
		fetched:    map[string]bool{},
	}
}

// latest returns the latest artifact of the branch and warns if it is not
// built from HEAD.
func (f *headArtifactFinder) latest(ctx context.Context) (artifact.Spec, error) {
	latest, err := actions.LatestArtifactFromBranch(f.client, f.service, f.branch)
	if err != nil {
		return artifact.Spec{}, err
	}
	// The comparison only informs, so failing it does not prevent the release.
	err = f.warnIfNotHead(ctx, latest)
	if err != nil {
		f.logger("Could not compare latest artifact %s with HEAD: %v\n", latest.ID, err)
	}
	return latest, nil
}

func (f *headArtifactFinder) warnIfNotHead(ctx context.Context, latest artifact.Spec) error {
	head, err := f.repository.HeadSHA(ctx)
	if err != nil {
		return err
	}
	relation, ahead, err := f.classify(ctx, head, latest.Application.SHA)
	if err != nil {
		return err
	}
	if relation == commitNewer {
		f.warnNewer(latest, head, ahead)
		return nil
	}
	if relation != commitCurrent {
		f.logger("Latest artifact %s is built from %s, not HEAD %s. "+
			"CI may still be building, or HEAD was built earlier; use --wait.\n",
			latest.ID, shortSHA(latest.Application.SHA), shortSHA(head))
	}
	return nil
}

// wait returns the artifact built from HEAD, or an artifact built from a
// descendant of HEAD, waiting up to timeout for either to be built.
func (f *headArtifactFinder) wait(ctx context.Context, timeout, pollInterval time.Duration) (artifact.Spec, error) {
	head, err := f.repository.HeadSHA(ctx)
	if err != nil {
		return artifact.Spec{}, err
	}
	// An unpushed HEAD is never built, so waiting for it would be futile.
	pushed, err := f.repository.IsPushed(ctx, head, f.branch)
	if err != nil || !pushed {
		return artifact.Spec{}, errors.Errorf("HEAD %s is not pushed to origin/%s", shortSHA(head), f.branch)
	}
	recent, found, err := f.recentArtifactOf(head)
	if err != nil {
		return artifact.Spec{}, err
	}
	if found {
		return recent, nil
	}
	return f.pollLatest(ctx, head, timeout, pollInterval)
}

// recentArtifactOf returns the newest of the recent artifacts of the branch
// built from head, if any, and warns if it is not the latest artifact.
func (f *headArtifactFinder) recentArtifactOf(head string) (artifact.Spec, bool, error) {
	recent, err := actions.ArtifactsFromBranch(f.client, f.service, f.branch, recentArtifactsCount)
	if err != nil {
		return artifact.Spec{}, false, err
	}
	for i, spec := range recent {
		if spec.Application.SHA != head {
			continue
		}
		if i > 0 {
			f.logger("Releasing %s built from HEAD %s; newer artifact %s exists on %s\n",
				spec.ID, shortSHA(head), recent[0].ID, f.branch)
		}
		return spec, true, nil
	}
	return artifact.Spec{}, false, nil
}

// pollLatest polls the latest artifact of the branch until it is releasable
// for head or timeout expires.
func (f *headArtifactFinder) pollLatest(
	ctx context.Context,
	head string,
	timeout, pollInterval time.Duration,
) (artifact.Spec, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var announced string
	for {
		latest, releasable, err := f.releasableLatest(ctx, head)
		if err != nil {
			return artifact.Spec{}, err
		}
		if releasable {
			return latest, nil
		}
		latestID := latest.ID
		if latestID == "" {
			latestID = "none"
		}
		if latestID != announced {
			f.logger("Waiting for artifact of %s (latest: %s)\n", shortSHA(head), latestID)
			announced = latestID
		}

		select {
		case <-ctx.Done():
			return artifact.Spec{}, ctx.Err()
		case <-timer.C:
			//nolint:revive // The message is shown to users as is.
			return artifact.Spec{}, errors.Errorf("Timed out after %s waiting for artifact of %s on %s; latest is %s. Check CI.",
				timeout, shortSHA(head), f.branch, latestID)
		case <-ticker.C:
		}
	}
}

// releasableLatest returns the latest artifact of the branch, if any, and
// whether it is releasable for head, i.e. built from head or from a descendant
// of head. It warns when releasing a descendant.
func (f *headArtifactFinder) releasableLatest(ctx context.Context, head string) (artifact.Spec, bool, error) {
	latest, err := actions.ArtifactsFromBranch(f.client, f.service, f.branch, 1)
	if err != nil || len(latest) == 0 {
		return artifact.Spec{}, false, err
	}
	relation, ahead, err := f.classify(ctx, head, latest[0].Application.SHA)
	if err != nil {
		return artifact.Spec{}, false, err
	}
	if relation == commitNewer {
		f.warnNewer(latest[0], head, ahead)
	}
	return latest[0], relation == commitCurrent || relation == commitNewer, nil
}

func (f *headArtifactFinder) warnNewer(spec artifact.Spec, head string, ahead int) {
	f.logger("Releasing %s built from %s, which is %s ahead of HEAD %s\n",
		spec.ID, shortSHA(spec.Application.SHA), commitCount(ahead), shortSHA(head))
}

// classify returns how sha relates to head and, if it is newer, by how many
// commits.
func (f *headArtifactFinder) classify(ctx context.Context, head, sha string) (commitRelation, int, error) {
	if sha == head {
		return commitCurrent, 0, nil
	}
	known, err := f.commitKnown(ctx, sha)
	if err != nil {
		return 0, 0, err
	}
	if !known {
		return commitDiverged, 0, nil
	}
	older, err := f.repository.IsAncestor(ctx, sha, head)
	if err != nil {
		return 0, 0, err
	}
	if older {
		return commitOlder, 0, nil
	}
	newer, err := f.repository.IsAncestor(ctx, head, sha)
	if err != nil {
		return 0, 0, err
	}
	if !newer {
		return commitDiverged, 0, nil
	}
	ahead, err := f.repository.CountCommits(ctx, head, sha)
	if err != nil {
		return 0, 0, err
	}
	return commitNewer, ahead, nil
}

// commitKnown reports whether sha is a commit in the repository, fetching the
// branch from origin the first time it is not.
func (f *headArtifactFinder) commitKnown(ctx context.Context, sha string) (bool, error) {
	if f.repository.CommitExists(ctx, sha) {
		return true, nil
	}
	if f.fetched[sha] {
		return false, nil
	}
	f.fetched[sha] = true
	err := f.repository.FetchBranch(ctx, f.branch)
	if err != nil {
		return false, err
	}
	return f.repository.CommitExists(ctx, sha), nil
}

// shortSHA abbreviates sha to the length Git abbreviates it to by default.
func shortSHA(sha string) string {
	const length = 7
	if len(sha) <= length {
		return sha
	}
	return sha[:length]
}

// commitCount returns n followed by "commit" or "commits".
func commitCount(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}

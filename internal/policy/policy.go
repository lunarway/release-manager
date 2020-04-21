package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	internalgit "github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
)

var (
	// ErrNotParsable indicates that a policies file could not be parsed against
	// the specification.
	ErrNotParsable = errors.New("policies not parsable")
	// ErrUnknownFields indicates that a policies file contains an unknown field.
	ErrUnknownFields = errors.New("policies contains unknown fields")
	// ErrNotFound indicates that policies are not found for a service.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates that polices are not compatible
	ErrConflict = errors.New("conflict")
)

type Service struct {
	Tracer tracing.Tracer
	Git    GitService

	MaxRetries                      int
	GlobalBranchRestrictionPolicies []BranchRestriction
}

type GitService interface {
	MasterPath() string
	Clone(context.Context, string) (*git.Repository, error)
	Commit(ctx context.Context, rootPath, changesPath, authorName, authorEmail, committerName, committerEmail, msg string) error
}

type Actor struct {
	Name  string
	Email string
}

// GetAutoReleases gets stored auto-release policies for service svc. If no
// policies are found a nil slice is returned.
func (s *Service) GetAutoReleases(ctx context.Context, svc, branch string) ([]AutoReleasePolicy, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.GetAutoReleases")
	defer span.Finish()
	policies, err := s.Get(ctx, svc)
	if err != nil {
		if errors.Cause(err) == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	var autoReleases []AutoReleasePolicy
	for i := range policies.AutoReleases {
		if policies.AutoReleases[i].Branch == branch {
			autoReleases = append(autoReleases, policies.AutoReleases[i])
		}
	}
	return autoReleases, nil
}

// Get gets stored policies for service svc. If no policies are stored
// ErrNotFound is returned. This method also returns globally configured
// policies along with the service specific ones.
func (s *Service) Get(ctx context.Context, svc string) (Policies, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.Get")
	defer span.Finish()

	// make sure policy directory exists
	policiesDir := path.Join(s.Git.MasterPath(), "policies")
	err := os.MkdirAll(policiesDir, os.ModePerm)
	if err != nil {
		return Policies{}, errors.WithMessagef(err, "make policies directory '%s'", policiesDir)
	}

	policies, err := s.servicePolicies(svc)
	if err != nil {
		// we will only return if an unknown error occoured and no global policies
		// are defined.
		if err != ErrNotFound || len(s.GlobalBranchRestrictionPolicies) == 0 {
			return Policies{}, err
		}
		policies = Policies{
			Service: svc,
		}
	}

	// merge global policies with local ones where globals take precedence
	policies.BranchRestrictions = mergeBranchRestrictions(ctx, svc, s.GlobalBranchRestrictionPolicies, policies.BranchRestrictions)
	log.WithContext(ctx).WithFields("globalPolicies", s.GlobalBranchRestrictionPolicies, "localPolicies", policies).Infof("Found %d policies", len(policies.BranchRestrictions)+len(policies.AutoReleases))

	// a policy file might exist, but if all policies have been removed from it
	// we can just act as if it didn't exist
	if !policies.HasPolicies() {
		return Policies{}, ErrNotFound
	}
	return policies, nil
}

// servicePolicies returns policies for a specific service. If no policy file is
// found ErrNotFound is returned.
func (s *Service) servicePolicies(svc string) (Policies, error) {
	// make sure policy directory exists
	policiesDir := path.Join(s.Git.MasterPath(), "policies")
	err := os.MkdirAll(policiesDir, os.ModePerm)
	if err != nil {
		return Policies{}, errors.WithMessagef(err, "make policies directory '%s'", policiesDir)
	}

	policiesPath := path.Join(policiesDir, fmt.Sprintf("%s.json", svc))
	policiesFile, err := os.OpenFile(policiesPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			return Policies{}, ErrNotFound
		}
		return Policies{}, errors.WithMessagef(err, "open policies in '%s'", policiesPath)
	}
	defer policiesFile.Close()

	policies, err := parse(policiesFile)
	if err != nil {
		return Policies{}, errors.WithMessagef(err, "parse policies in '%s'", policiesPath)
	}
	return policies, nil
}

func mergeBranchRestrictions(ctx context.Context, svc string, global, local []BranchRestriction) []BranchRestriction {
	if len(global) == 0 {
		return local
	}
	branchRestrictions := append([]BranchRestriction(nil), global...)

	// copy all local restrictions over that does not conflict with the global
	// one
	for _, localRestriction := range local {
		if conflictingBranchRestriction(ctx, svc, global, localRestriction) {
			continue
		}
		branchRestrictions = append(branchRestrictions, localRestriction)
	}
	return branchRestrictions
}

func conflictingBranchRestriction(ctx context.Context, svc string, global []BranchRestriction, localRestriction BranchRestriction) bool {
	for _, globalRestriction := range global {
		if globalRestriction.Environment == localRestriction.Environment && globalRestriction.BranchRegex != localRestriction.BranchRegex {
			log.WithContext(ctx).WithFields("global", globalRestriction, "local", localRestriction).Errorf("Global and local branch restriction policies conflict for service '%s': local policy dropped", svc)
			return true
		}
	}
	return false
}

// ApplyAutoRelease applies an auto-release policy for service svc from branch
// to environment env.
func (s *Service) ApplyAutoRelease(ctx context.Context, actor Actor, svc, branch, env string) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.ApplyAutoRelease")
	defer span.Finish()

	ok, err := s.CanRelease(ctx, svc, branch, env)
	if err != nil {
		return "", errors.WithMessage(err, "validate release policies")
	}
	if !ok {
		return "", ErrConflict
	}

	commitMsg := internalgit.PolicyUpdateApplyCommitMessage(env, svc, "auto-release")
	var policyID string
	err = s.updatePolicies(ctx, actor, svc, commitMsg, func(p *Policies) {
		policyID = p.SetAutoRelease(branch, env)
	})
	if err != nil {
		return "", err
	}
	return policyID, nil
}

// Delete deletes policies by ID for service svc.
func (s *Service) Delete(ctx context.Context, actor Actor, svc string, ids []string) (int, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.Delete")
	defer span.Finish()
	commitMsg := internalgit.PolicyUpdateDeleteCommitMessage(svc)
	var deleted int
	err := s.updatePolicies(ctx, actor, svc, commitMsg, func(p *Policies) {
		deleted = p.Delete(ids...)
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Service) updatePolicies(ctx context.Context, actor Actor, svc, commitMsg string, f func(p *Policies)) error {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.updatePolicies")
	defer span.Finish()
	return try.Do(ctx, s.Tracer, s.MaxRetries, func(ctx context.Context, attempt int) (bool, error) {
		configRepoPath, close, err := internalgit.TempDirAsync(ctx, s.Tracer, "k8s-config-notify")
		if err != nil {
			return true, err
		}
		defer close(ctx)

		logger := log.WithContext(ctx)

		// read part of this code is the same as the Get function but differs in the
		// file flags used. This is to avoid opening and closing to file multiple
		// times during the operation.
		logger.Debugf("internal/policy: clone config repository")
		_, err = s.Git.Clone(ctx, configRepoPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("clone to '%s'", configRepoPath))
		}

		// make sure policy directory exists
		logger.Debugf("internal/policy: ensure policies directory")
		policiesDir := path.Join(configRepoPath, "policies")
		err = os.MkdirAll(policiesDir, os.ModePerm)
		if err != nil {
			return true, errors.WithMessagef(err, "make policies directory '%s'", policiesDir)
		}

		policiesPath := path.Join(policiesDir, fmt.Sprintf("%s.json", svc))
		logger.Debugf("internal/policy: open policies file '%s'", policiesPath)
		policiesFile, err := os.OpenFile(policiesPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			return true, errors.WithMessagef(err, "open policies in '%s'", policiesPath)
		}
		defer policiesFile.Close()

		// read existing policies
		logger.Debugf("internal/policy: parse policies file '%s'", policiesPath)
		policies, err := parse(policiesFile)
		if err != nil {
			return true, errors.WithMessagef(err, "parse policies in '%s'", policiesPath)
		}
		logger.Debugf("internal/policy: parseed policy: %+v", policies)

		policies.Service = svc
		f(&policies)

		// store file

		// truncate and reset the offset of the file before writing to it
		// to overwrite the contents
		err = policiesFile.Truncate(0)
		if err != nil {
			return true, errors.WithMessagef(err, "truncate file '%s'", policiesPath)
		}
		_, err = policiesFile.Seek(0, 0)
		if err != nil {
			return true, errors.WithMessagef(err, "reset seek on '%s'", policiesPath)
		}
		logger.Debugf("internal/policy: persist policies file '%s'", policiesPath)
		err = persist(policiesFile, policies)
		if err != nil {
			return true, errors.WithMessagef(err, "write policies in '%s'", policiesPath)
		}

		// commit changes
		logger.Debugf("internal/policy: commit policies file '%s'", policiesPath)
		err = s.Git.Commit(ctx, configRepoPath, path.Join(".", "policies"), actor.Name, actor.Email, actor.Name, actor.Email, commitMsg)
		if err != nil {
			// indicates that the applied policy was already set
			if errors.Cause(err) == internalgit.ErrNothingToCommit {
				return true, nil
			}
			return false, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", policiesPath))
		}
		return true, nil
	})
}

func parse(r io.Reader) (Policies, error) {
	var p Policies
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&p)
	if err != nil {
		if err == io.EOF {
			return Policies{}, nil
		}
		_, ok := err.(*json.SyntaxError)
		if ok {
			return Policies{}, ErrNotParsable
		}
		// there is no other way to detect this error type unfortunately
		// https://github.com/golang/go/blob/277609f844ed9254d25e975f7cf202d042beecc6/src/encoding/json/decode.go#L739
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			return Policies{}, errors.WithMessagef(ErrUnknownFields, "%v", err)
		}
		return Policies{}, errors.WithMessage(err, "decode policy as json")
	}
	return p, nil
}

func persist(w io.Writer, p Policies) error {
	encode := json.NewEncoder(w)
	encode.SetIndent("", "  ")
	err := encode.Encode(p)
	if err != nil {
		return err
	}
	return nil
}

type Policies struct {
	Service            string              `json:"service,omitempty"`
	AutoReleases       []AutoReleasePolicy `json:"autoReleases,omitempty"`
	BranchRestrictions []BranchRestriction `json:"branchRestrictions,omitempty"`
}

type AutoReleasePolicy struct {
	ID          string `json:"id,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// HasPolicies returns whether any policies are applied.
func (p *Policies) HasPolicies() bool {
	return len(p.AutoReleases) != 0 || len(p.BranchRestrictions) != 0
}

// SetAutoRelease sets an auto-release policy for specified branch and
// environment.
//
// If an auto-release policy exists for the same environment it is overwritten.
func (p *Policies) SetAutoRelease(branch, env string) string {
	id := fmt.Sprintf("auto-release-%s-%s", branch, env)
	newPolicy := AutoReleasePolicy{
		ID:          id,
		Branch:      branch,
		Environment: env,
	}
	newPolicies := make([]AutoReleasePolicy, len(p.AutoReleases))
	var replaced bool
	for i, policy := range p.AutoReleases {
		if policy.Environment == env {
			newPolicies[i] = newPolicy
			replaced = true
			continue
		}
		newPolicies[i] = p.AutoReleases[i]
	}
	if !replaced {
		newPolicies = append(newPolicies, newPolicy)
	}
	p.AutoReleases = newPolicies
	return id
}

// Delete deletes any policies with a matching id.
func (p *Policies) Delete(ids ...string) int {
	var deleted int
	for _, id := range ids {
		var filtered []AutoReleasePolicy
		for i := range p.AutoReleases {
			if p.AutoReleases[i].ID != id {
				filtered = append(filtered, p.AutoReleases[i])
				continue
			}
			deleted++
		}
		p.AutoReleases = filtered

		var filteredBranchRestrictions []BranchRestriction
		for i := range p.BranchRestrictions {
			if p.BranchRestrictions[i].ID != id {
				filteredBranchRestrictions = append(filteredBranchRestrictions, p.BranchRestrictions[i])
				continue
			}
			deleted++
		}
		p.BranchRestrictions = filteredBranchRestrictions
	}
	return deleted
}

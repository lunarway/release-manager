package policy

import (
	"context"
	"fmt"
	"regexp"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

type BranchRestrictor struct {
	ID          string `json:"id,omitempty"`
	BranchRegex string `json:"branchRegex,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// ApplyBranchRestrictor applies a branch-restrictor policy for service svc to
// environment env with regular expression branchRegex.
func (s *Service) ApplyBranchRestrictor(ctx context.Context, actor Actor, svc, branchRegex, env string) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.ApplyBranchRestrictor")
	defer span.Finish()

	// validate branch regular expression before storring
	re, err := regexp.Compile(branchRegex)
	if err != nil {
		return "", errors.WithMessage(err, "branch regex not valid")
	}

	// ensure no auto release policies will conflict with this one
	policies, err := s.Get(ctx, svc)
	if err != nil && errors.Cause(err) != ErrNotFound {
		return "", err
	}
	for _, policy := range policies.AutoReleases {
		if re.MatchString(policy.Branch) {
			return "", ErrConflict
		}
	}

	commitMsg := git.PolicyUpdateApplyCommitMessage(env, svc, "branch-restrictor")
	var policyID string
	err = s.updatePolicies(ctx, actor, svc, commitMsg, func(p *Policies) {
		policyID = p.SetBranchRestrictor(branchRegex, env)
	})
	if err != nil {
		return "", err
	}
	return policyID, nil
}

// CanRelease returns whether service svc's branch can be released to env.
func (s *Service) CanRelease(ctx context.Context, svc, branch, env string) (bool, error) {
	log.WithContext(ctx).Infof("Verifying whether %s on branch %s can be released to %s", svc, branch, env)
	span, ctx := s.Tracer.FromCtx(ctx, "policy.CanRelease")
	defer span.Finish()
	policies, err := s.Get(ctx, svc)
	if err != nil {
		if errors.Cause(err) == ErrNotFound {
			return true, nil
		}
		return false, err
	}
	log.WithContext(ctx).WithFields("policies", policies).Infof("Found %d restrictors", len(policies.BranchRestrictors))
	span, _ = s.Tracer.FromCtx(ctx, "policy.canRelease")
	defer span.Finish()
	return canRelease(ctx, policies, branch, env)
}

func canRelease(ctx context.Context, policies Policies, branch, env string) (bool, error) {
	for _, policy := range policies.BranchRestrictors {
		if policy.Environment != env {
			continue
		}
		r, err := regexp.Compile(policy.BranchRegex)
		if err != nil {
			return false, errors.WithMessage(err, "branch regex not valid regular expression")
		}
		if r.MatchString(branch) {
			return true, nil
		}
		return false, nil
	}
	return true, nil
}

// SetBranchRestrictor sets a branch-restrictor policy for specified environment
// and branch regex.
//
// If a policy exists for the same environment it is overwritten.
func (p *Policies) SetBranchRestrictor(branchRegex string, env string) string {
	id := fmt.Sprintf("branch-restrictor-%s", env)
	newPolicy := BranchRestrictor{
		ID:          id,
		BranchRegex: branchRegex,
		Environment: env,
	}
	newPolicies := make([]BranchRestrictor, len(p.BranchRestrictors))
	var replaced bool
	for i, policy := range p.BranchRestrictors {
		if policy.Environment == env {
			newPolicies[i] = newPolicy
			replaced = true
			continue
		}
		newPolicies[i] = p.BranchRestrictors[i]
	}
	if !replaced {
		newPolicies = append(newPolicies, newPolicy)
	}
	p.BranchRestrictors = newPolicies
	return id
}

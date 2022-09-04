package policy

import (
	"context"
	"fmt"
	"regexp"

	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/pkg/errors"
)

type BranchRestriction struct {
	ID          string `json:"id,omitempty"`
	BranchRegex string `json:"branchRegex,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// ApplyBranchRestriction applies a branch-restriction policy for service svc to
// environment env with regular expression branchRegex.
func (s *Service) ApplyBranchRestriction(ctx context.Context, actor Actor, svc, branchRegex, env string) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "policy.ApplyBranchRestriction")
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
		if policy.Environment == env && !re.MatchString(policy.Branch) {
			return "", errors.WithMessagef(ErrConflict, "conflict with %s", policy.ID)
		}
	}

	// check that it does not conflict with a global policy
	if conflictingBranchRestriction(ctx, s.Logger, svc, s.GlobalBranchRestrictionPolicies, BranchRestriction{
		BranchRegex: branchRegex,
		Environment: env,
	}) {
		return "", errors.WithMessagef(ErrConflict, "conflicts with global policy")
	}

	commitMsg := commitinfo.PolicyUpdateApplyCommitMessage(env, svc, "branch-restriction")
	var policyID string
	err = s.updatePolicies(ctx, actor, svc, commitMsg, func(p *Policies) {
		policyID = p.SetBranchRestriction(branchRegex, env)
	})
	if err != nil {
		return "", err
	}
	return policyID, nil
}

// CanRelease returns whether service svc's branch can be released to env.
func (s *Service) CanRelease(ctx context.Context, svc, branch, env string) (bool, error) {
	s.Logger.WithContext(ctx).Infof("Verifying whether %s on branch %s can be released to %s", svc, branch, env)
	span, ctx := s.Tracer.FromCtx(ctx, "policy.CanRelease")
	defer span.Finish()
	policies, err := s.Get(ctx, svc)
	if err != nil {
		if errors.Cause(err) == ErrNotFound {
			return true, nil
		}
		return false, err
	}
	s.Logger.WithContext(ctx).WithFields("policies", policies).Infof("Found %d restrictions", len(policies.BranchRestrictions))
	span, _ = s.Tracer.FromCtx(ctx, "policy.canRelease")
	defer span.Finish()
	return canRelease(ctx, policies, branch, env)
}

func canRelease(ctx context.Context, policies Policies, branch, env string) (bool, error) {
	for _, policy := range policies.BranchRestrictions {
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

// SetBranchRestriction sets a branch-restriction policy for specified environment
// and branch regex.
//
// If a policy exists for the same environment it is overwritten.
func (p *Policies) SetBranchRestriction(branchRegex string, env string) string {
	id := fmt.Sprintf("branch-restriction-%s", env)
	newPolicy := BranchRestriction{
		ID:          id,
		BranchRegex: branchRegex,
		Environment: env,
	}
	newPolicies := make([]BranchRestriction, len(p.BranchRestrictions))
	var replaced bool
	for i, policy := range p.BranchRestrictions {
		if policy.Environment == env {
			newPolicies[i] = newPolicy
			replaced = true
			continue
		}
		newPolicies[i] = p.BranchRestrictions[i]
	}
	if !replaced {
		newPolicies = append(newPolicies, newPolicy)
	}
	p.BranchRestrictions = newPolicies
	return id
}

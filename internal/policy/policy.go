package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

var (
	configRepoPath = path.Join(".tmp", "k8s-config-policies")
)

var (
	// ErrNotParsable indicates that a policies file could not be parsed against
	// the specification.
	ErrNotParsable = errors.New("policies not parsable")
	// ErrUnknownFields indicates that a policies file contains an unknown field.
	ErrUnknownFields = errors.New("policies contains unknown fields")
	// ErrNotFound indicates that policies are not found for a service.
	ErrNotFound = errors.New("not found")
)

// GetAutoReleases gets stored auto-release policies for service svc. If no
// policies are found a nil slice is returned.
func GetAutoReleases(ctx context.Context, configRepoURL, sshPrivateKeyPath string, svc, branch string) ([]AutoReleasePolicy, error) {
	policies, err := Get(ctx, configRepoURL, sshPrivateKeyPath, svc)
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

// Get gets stored policies for service svc. If no policies are stored ErrNotFound is returned.
func Get(ctx context.Context, configRepoURL, sshPrivateKeyPath string, svc string) (Policies, error) {
	_, err := git.CloneDepth(ctx, configRepoURL, configRepoPath, sshPrivateKeyPath, 1)
	if err != nil {
		return Policies{}, errors.WithMessage(err, fmt.Sprintf("clone to path '%s'", configRepoPath))
	}

	// make sure policy directory exists
	policiesDir := path.Join(configRepoPath, "policies")
	err = os.MkdirAll(policiesDir, os.ModePerm)
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
		policiesFile.Close()
		return Policies{}, errors.WithMessagef(err, "parse policies in '%s'", policiesPath)
	}
	// a policy file might exist, but if all policies have been removed from it
	// we can just act as if it didn't exist
	if !policies.HasPolicies() {
		return Policies{}, ErrNotFound
	}
	return policies, nil
}

// ApplyAutoRelease applies an auto-release policy for service svc from branch
// to environment env.
func ApplyAutoRelease(ctx context.Context, configRepoURL, sshPrivateKeyPath string, svc, branch, env, committerName, committerEmail string) (string, error) {
	commitMsg := fmt.Sprintf("[%s] policy update: set auto-release from '%s' to '%s'", svc, branch, env)
	var policyID string
	err := updatePolicies(ctx, configRepoURL, sshPrivateKeyPath, svc, commitMsg, committerName, committerEmail, func(p *Policies) {
		policyID = p.SetAutoRelease(branch, env)
	})
	if err != nil {
		return "", err
	}
	return policyID, nil
}

// Delete deletes policies by ID for service svc.
func Delete(ctx context.Context, configRepoURL, sshPrivateKeyPath string, svc string, ids []string, committerName, committerEmail string) (int, error) {
	commitMsg := fmt.Sprintf("[%s] policy update: delete policies", svc)
	var deleted int
	err := updatePolicies(ctx, configRepoURL, sshPrivateKeyPath, svc, commitMsg, committerName, committerEmail, func(p *Policies) {
		deleted = p.Delete(ids...)
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func updatePolicies(ctx context.Context, configRepoURL, sshPrivateKeyPath, svc, commitMsg, committerName, committerEmail string, f func(p *Policies)) error {
	log.Debugf("internal/policy: clone config repository")
	repo, err := git.CloneDepth(ctx, configRepoURL, configRepoPath, sshPrivateKeyPath, 1)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, configRepoPath))
	}

	// make sure policy directory exists
	log.Debugf("internal/policy: ensure policies directory")
	err = os.MkdirAll(path.Join(configRepoPath, "policies"), os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, "make policies directory")
	}

	policiesPath := path.Join(configRepoPath, "policies", fmt.Sprintf("%s.json", svc))
	log.Debugf("internal/policy: open policies file '%s'", policiesPath)
	policiesFile, err := os.OpenFile(policiesPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "open policies in '%s'", policiesPath)
	}
	defer policiesFile.Close()

	// read existing policies
	log.Debugf("internal/policy: parse policies file '%s'", policiesPath)
	policies, err := parse(policiesFile)
	if err != nil {
		return errors.WithMessagef(err, "parse policies in '%s'", policiesPath)
	}
	log.Debugf("internal/policy: parseed policy: %+v", policies)

	policies.Service = svc
	f(&policies)

	// store file

	// truncate and reset the offset of the file before writing to it
	// to overwrite the contents
	err = policiesFile.Truncate(0)
	if err != nil {
		return errors.WithMessagef(err, "truncate file '%s'", policiesPath)
	}
	_, err = policiesFile.Seek(0, 0)
	if err != nil {
		return errors.WithMessagef(err, "reset seek on '%s'", policiesPath)
	}
	log.Debugf("internal/policy: persist policies file '%s'", policiesPath)
	err = persist(policiesFile, policies)
	if err != nil {
		return errors.WithMessagef(err, "write policies in '%s'", policiesPath)
	}

	// commit changes
	log.Debugf("internal/policy: commit policies file '%s'", policiesPath)
	err = git.Commit(ctx, repo, path.Join(".", "policies"), committerName, committerEmail, committerName, committerEmail, commitMsg, sshPrivateKeyPath)
	if err != nil {
		// indicates that the applied policy was already set
		if err == git.ErrNothingToCommit {
			return nil
		}
		return errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", policiesPath))
	}
	return nil
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
	Service      string              `json:"service,omitempty"`
	AutoReleases []AutoReleasePolicy `json:"autoReleases,omitempty"`
}

type AutoReleasePolicy struct {
	ID          string `json:"id,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// HasPolicies returns whether any policies are applied.
func (p *Policies) HasPolicies() bool {
	return len(p.AutoReleases) != 0
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
	}
	return deleted
}

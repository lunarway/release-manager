package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp/syntax"
	"strings"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
)

type policyPatchPath struct {
	segments []string
}

func newPolicyPatchPath(r *http.Request) (policyPatchPath, bool) {
	p := policyPatchPath{
		segments: strings.Split(r.URL.Path, "/"),
	}
	if len(p.segments) < 3 {
		return policyPatchPath{}, false
	}
	return p, true
}

func (p *policyPatchPath) PolicyType() string {
	return p.segments[2]
}

func policy(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			ctx := r.Context()
			p, ok := newPolicyPatchPath(r)
			if !ok {
				log.WithContext(ctx).Errorf("Could not parse PATCH policy path: %s", r.URL.Path)
				Error(w, "not found", http.StatusNotFound)
				return
			}
			switch p.PolicyType() {
			case "auto-release":
				applyAutoReleasePolicy(payload, policySvc)(w, r)
			case "branch-restriction":
				applyBranchRestrictorPolicy(payload, policySvc)(w, r)
			default:
				log.WithContext(ctx).Errorf("apply policy not found: %+v", p)
				notFound(w)
			}
		case http.MethodGet:
			listPolicies(payload, policySvc)(w, r)
		case http.MethodDelete:
			deletePolicies(payload, policySvc)(w, r)
		default:
			Error(w, "not found", http.StatusNotFound)
		}
	}
}

func applyAutoReleasePolicy(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.ApplyAutoReleasePolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: policy: apply auto-release: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		if emptyString(req.Service) {
			requiredFieldError(w, "service")
			return
		}
		if emptyString(req.Branch) {
			requiredFieldError(w, "branch")
			return
		}
		if emptyString(req.Environment) {
			requiredFieldError(w, "environment")
			return
		}
		if emptyString(req.CommitterName) {
			requiredFieldError(w, "committerName")
			return
		}
		if emptyString(req.CommitterEmail) {
			requiredFieldError(w, "committerEmail")
			return
		}
		logger = logger.WithFields("service", req.Service, "req", req)
		logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release policy started", req.Service, req.Branch, req.Environment)
		id, err := policySvc.ApplyAutoRelease(ctx, policyinternal.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Service, req.Branch, req.Environment)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release cancelled", req.Service, req.Branch, req.Environment)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case policyinternal.ErrConflict:
				logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release rejected: conflicts with another policy: %v", req.Service, req.Branch, req.Environment, err)
				Error(w, fmt.Sprintf("policy conflicts with another policy"), http.StatusServiceUnavailable)
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': %v", req.Service, req.Branch, req.Environment, err)
				Error(w, fmt.Sprintf("could not apply policy right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			default:
				logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release failed: %v", req.Service, req.Branch, req.Environment, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = payload.encodeResponse(ctx, w, httpinternal.ApplyPolicyResponse{
			ID:          id,
			Service:     req.Service,
			Branch:      req.Branch,
			Environment: req.Environment,
		})
		if err != nil {
			logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release: marshal response failed: %v", req.Service, req.Branch, req.Environment, err)
		}
	}
}

func applyBranchRestrictorPolicy(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.ApplyBranchRestrictorPolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: policy: apply: branch-restriction: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		if emptyString(req.Service) {
			requiredFieldError(w, "service")
			return
		}
		if emptyString(req.Environment) {
			requiredFieldError(w, "environment")
			return
		}
		if emptyString(req.BranchMatcher) {
			requiredFieldError(w, "branch matcher")
			return
		}
		if emptyString(req.CommitterName) {
			requiredFieldError(w, "committerName")
			return
		}
		if emptyString(req.CommitterEmail) {
			requiredFieldError(w, "committerEmail")
			return
		}
		logger = logger.WithFields("service", req.Service, "req", req)
		logger.Infof("http: policy: apply: service '%s' branch matcher '%s' environment '%s': apply branch-restriction policy started", req.Service, req.BranchMatcher, req.Environment)
		id, err := policySvc.ApplyBranchRestrictor(ctx, policyinternal.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Service, req.BranchMatcher, req.Environment)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: policy: apply: service '%s' branch matcher '%s' environment '%s': apply branch-restriction cancelled", req.Service, req.BranchMatcher, req.Environment)
				cancelled(w)
				return
			}
			var regexErr *syntax.Error
			if errors.As(err, &regexErr) {
				logger.Infof("http: policy: apply: service '%s' branch matcher '%s' environment '%s': apply branch-restrction: invalid branch matcher: %v", req.Service, req.BranchMatcher, req.Environment, err)
				Error(w, fmt.Sprintf("branch matcher not valid: %v", regexErr), http.StatusBadRequest)
				return
			}
			switch errorCause(err) {
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: apply: service '%s' branch matcher '%s' environment '%s': apply branch-restrction: %v", req.Service, req.BranchMatcher, req.Environment, err)
				Error(w, fmt.Sprintf("could not apply policy right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			default:
				logger.Errorf("http: policy: apply: service '%s' branch matcher '%s' environment '%s': apply branch-restriction failed: %v", req.Service, req.BranchMatcher, req.Environment, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = payload.encodeResponse(ctx, w, httpinternal.ApplyBranchRestrictorPolicyResponse{
			ID:            id,
			Service:       req.Service,
			BranchMatcher: req.BranchMatcher,
			Environment:   req.Environment,
		})
		if err != nil {
			logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply branch-restriction: marshal response failed: %v", req.Service, req.BranchMatcher, req.Environment, err)
		}
	}
}

func listPolicies(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		service := values.Get("service")
		if emptyString(service) {
			requiredQueryError(w, "service")
			return
		}

		ctx := r.Context()
		logger := log.WithContext(ctx).WithFields("service", service)
		policies, err := policySvc.Get(ctx, service)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: policy: list: service '%s': get policies cancelled", service)
				cancelled(w)
				return
			}
			if errorCause(err) == policyinternal.ErrNotFound {
				Error(w, "no policies exist", http.StatusNotFound)
				return
			}
			logger.Errorf("http: policy: list: service '%s': get policies failed: %v", service, err)
			unknownError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.ListPoliciesResponse{
			Service:           policies.Service,
			AutoReleases:      mapAutoReleasePolicies(policies.AutoReleases),
			BranchRestrictors: mapBranchRestrictorPolicies(policies.BranchRestrictors),
		})
		if err != nil {
			logger.Errorf("http: policy: list: service '%s': marshal response failed: %v", service, err)
		}
	}
}

func mapAutoReleasePolicies(policies []policyinternal.AutoReleasePolicy) []httpinternal.AutoReleasePolicy {
	h := make([]httpinternal.AutoReleasePolicy, len(policies))
	for i, p := range policies {
		h[i] = httpinternal.AutoReleasePolicy{
			ID:          p.ID,
			Branch:      p.Branch,
			Environment: p.Environment,
		}
	}
	return h
}

func mapBranchRestrictorPolicies(policies []policyinternal.BranchRestrictor) []httpinternal.BranchRestrictorPolicy {
	h := make([]httpinternal.BranchRestrictorPolicy, len(policies))
	for i, p := range policies {
		h[i] = httpinternal.BranchRestrictorPolicy{
			ID:            p.ID,
			Environment:   p.Environment,
			BranchMatcher: p.BranchMatcher,
		}
	}
	return h
}

func deletePolicies(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.DeletePolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: policy: delete: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		if emptyString(req.Service) {
			requiredFieldError(w, "service")
			return
		}
		if emptyString(req.CommitterName) {
			requiredFieldError(w, "committerName")
			return
		}
		if emptyString(req.CommitterEmail) {
			requiredFieldError(w, "committerEmail")
			return
		}
		ids := filterEmptyStrings(req.PolicyIDs)
		if len(ids) == 0 {
			Error(w, "no policy ids suplied", http.StatusBadRequest)
			return
		}

		logger = logger.WithFields("service", req.Service, "req", req)

		deleted, err := policySvc.Delete(ctx, policyinternal.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Service, ids)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Errorf("http: policy: delete: service '%s' ids %v: delete cancelled", req.Service, ids)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case policyinternal.ErrNotFound:
				Error(w, "no policies exist", http.StatusNotFound)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: delete: service '%s' ids %v: %v", req.Service, ids, err)
				Error(w, fmt.Sprintf("could not delete policy right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			default:
				logger.Errorf("http: policy: delete: service '%s' ids %v: delete failed: %v", req.Service, ids, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DeletePolicyResponse{
			Service: req.Service,
			Count:   deleted,
		})
		if err != nil {
			logger.Errorf("http: policy: delete: service '%s' ids %v: marshal response failed: %v", req.Service, ids, err)
		}
	}
}

func filterEmptyStrings(ss []string) []string {
	var f []string
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		f = append(f, s)
	}
	return f
}

func emptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

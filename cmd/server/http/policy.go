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

		if !req.Validate(w) {
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
				httpinternal.Error(w, "policy conflicts with another policy", http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': %v", req.Service, req.Branch, req.Environment, err)
				httpinternal.Error(w, "could not apply policy right now. Please try again in a moment.", http.StatusServiceUnavailable)
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

func applyBranchRestrictionPolicy(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.ApplyBranchRestrictionPolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: policy: apply: branch-restriction: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		if !req.Validate(w) {
			return
		}

		logger = logger.WithFields("service", req.Service, "req", req)
		logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction policy started", req.Service, req.BranchRegex, req.Environment)
		id, err := policySvc.ApplyBranchRestriction(ctx, policyinternal.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Service, req.BranchRegex, req.Environment)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction cancelled", req.Service, req.BranchRegex, req.Environment)
				cancelled(w)
				return
			}
			var regexErr *syntax.Error
			if errors.As(err, &regexErr) {
				logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction: invalid branch regex: %v", req.Service, req.BranchRegex, req.Environment, err)
				httpinternal.Error(w, fmt.Sprintf("branch regex not valid: %v", regexErr), http.StatusBadRequest)
				return
			}
			switch errorCause(err) {
			case policyinternal.ErrConflict:
				logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction rejected: conflicts with another policy: %v", req.Service, req.BranchRegex, req.Environment, err)
				httpinternal.Error(w, fmt.Sprintf("policy conflicts with another policy"), http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction: %v", req.Service, req.BranchRegex, req.Environment, err)
				httpinternal.Error(w, fmt.Sprintf("could not apply policy right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			default:
				logger.Errorf("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction failed: %v", req.Service, req.BranchRegex, req.Environment, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = payload.encodeResponse(ctx, w, httpinternal.ApplyBranchRestrictionPolicyResponse{
			ID:          id,
			Service:     req.Service,
			BranchRegex: req.BranchRegex,
			Environment: req.Environment,
		})
		if err != nil {
			logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply branch-restriction: marshal response failed: %v", req.Service, req.BranchRegex, req.Environment, err)
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
				httpinternal.Error(w, "no policies exist", http.StatusNotFound)
				return
			}
			logger.Errorf("http: policy: list: service '%s': get policies failed: %v", service, err)
			unknownError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.ListPoliciesResponse{
			Service:            policies.Service,
			AutoReleases:       mapAutoReleasePolicies(policies.AutoReleases),
			BranchRestrictions: mapBranchRestrictionPolicies(policies.BranchRestrictions),
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

func mapBranchRestrictionPolicies(policies []policyinternal.BranchRestriction) []httpinternal.BranchRestrictionPolicy {
	h := make([]httpinternal.BranchRestrictionPolicy, len(policies))
	for i, p := range policies {
		h[i] = httpinternal.BranchRestrictionPolicy{
			ID:          p.ID,
			Environment: p.Environment,
			BranchRegex: p.BranchRegex,
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

		if !req.Validate(w) {
			return
		}

		ids := filterEmptyStrings(req.PolicyIDs)
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
				httpinternal.Error(w, "no policies exist", http.StatusNotFound)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: policy: delete: service '%s' ids %v: %v", req.Service, ids, err)
				httpinternal.Error(w, fmt.Sprintf("could not delete policy right now. Please try again in a moment."), http.StatusServiceUnavailable)
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

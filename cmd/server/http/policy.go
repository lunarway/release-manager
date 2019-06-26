package http

import (
	"context"
	"net/http"
	"strings"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
)

func policy(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			// only auto-release policies are available so no other validtion is required here
			applyAutoReleasePolicy(payload, policySvc)(w, r)
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
		var req httpinternal.ApplyAutoReleasePolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			log.Errorf("http: policy: apply: decode request body failed: %v", err)
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
		logger := log.WithFields("service", req.Service, "req", req)
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
			logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release failed: %v", req.Service, req.Branch, req.Environment, err)
			unknownError(w)
			return
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

func listPolicies(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		service := values.Get("service")
		if emptyString(service) {
			requiredQueryError(w, "service")
			return
		}

		logger := log.WithFields("service", service)
		ctx := r.Context()
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
			Service:      policies.Service,
			AutoReleases: mapAutoReleasePolicies(policies.AutoReleases),
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

func deletePolicies(payload *payload, policySvc *policyinternal.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req httpinternal.DeletePolicyRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			log.Errorf("http: policy: delete: decode request body failed: %v", err)
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

		logger := log.WithFields("service", req.Service, "req", req)

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
			if errorCause(err) == policyinternal.ErrNotFound {
				Error(w, "no policies exist", http.StatusNotFound)
				return
			}
			logger.Errorf("http: policy: delete: service '%s' ids %v: delete failed: %v", req.Service, ids, err)
			unknownError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DeletePolicyResponse{
			Service: req.Service,
			Count:   deleted,
		})
		if err != nil {
			log.Errorf("http: policy: delete: service '%s' ids %v: marshal response failed: %v", req.Service, ids, err)
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

package http

import (
	"encoding/json"
	"net/http"
	"strings"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/pkg/errors"
)

func policy(configRepo, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid := validateToken(r.Header.Get("Authorization"), "HAMCTL_AUTH_TOKEN")
		if !valid {
			Error(w, "not authorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			// TODO: detect what policy type is added based on path or payload
			applyAutoReleasePolicy(configRepo, sshPrivateKeyPath)(w, r)
			return
		case http.MethodGet:
			listPolicies(configRepo, sshPrivateKeyPath)(w, r)
			return
		case http.MethodDelete:
			deletePolicies(configRepo, sshPrivateKeyPath)(w, r)
		default:
			Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}

func applyAutoReleasePolicy(configRepo, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.ApplyAutoReleasePolicyRequest
		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("Decode request body failed: %v", err)
			Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if len(req.Service) == 0 {
			requiredFieldError(w, "service")
			return
		}
		if len(req.Branch) == 0 {
			requiredFieldError(w, "branch")
			return
		}
		if len(req.Environment) == 0 {
			requiredFieldError(w, "environment")
			return
		}
		if len(req.CommitterName) == 0 {
			requiredFieldError(w, "committerName")
			return
		}
		if len(req.CommitterEmail) == 0 {
			requiredFieldError(w, "committerEmail")
			return
		}

		log.Infof("http apply auto-release policy started: service '%s' branch '%s' environment '%s'", req.Service, req.Branch, req.Environment)
		id, err := policyinternal.ApplyAutoRelease(r.Context(), configRepo, sshPrivateKeyPath, req.Service, req.Branch, req.Environment, req.CommitterName, req.CommitterEmail)
		if err != nil {
			log.Errorf("http apply auto-release policy failed: config repo '%s' service '%s' branch '%s' environment '%s': %v", configRepo, req.Service, req.Branch, req.Environment, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(httpinternal.ApplyPolicyResponse{
			ID:          id,
			Service:     req.Service,
			Branch:      req.Branch,
			Environment: req.Environment,
		})
		if err != nil {
			log.Errorf("http apply auto-release policy failed: config repo '%s' service '%s' branch '%s' environment '%s': encode response: %v", configRepo, req.Service, req.Branch, req.Environment, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}
	}
}

func listPolicies(configRepo, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		service := values.Get("service")
		if len(service) == 0 {
			requiredQueryError(w, "service")
			return
		}

		policies, err := policyinternal.Get(r.Context(), configRepo, sshPrivateKeyPath, service)
		if err != nil {
			if errors.Cause(err) == policyinternal.ErrNotFound {
				Error(w, "no policies exist", http.StatusNotFound)
				return
			}
			log.Errorf("http list policies failed: config repo '%s' service '%s': %v", configRepo, service, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(httpinternal.ListPoliciesResponse{
			Service:      policies.Service,
			AutoReleases: mapAutoReleasePolicies(policies.AutoReleases),
		})
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

func deletePolicies(configRepo, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.DeletePolicyRequest
		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("Decode request body failed: %v", err)
			Error(w, "invalid payload", http.StatusBadRequest)
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

		deleted, err := policyinternal.Delete(r.Context(), configRepo, sshPrivateKeyPath, req.Service, ids, req.CommitterName, req.CommitterEmail)
		if err != nil {
			if errors.Cause(err) == policyinternal.ErrNotFound {
				Error(w, "no policies exist", http.StatusNotFound)
				return
			}
			log.Errorf("http list policies failed: config repo '%s' service '%s': %v", configRepo, req.Service, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(httpinternal.DeletePolicyResponse{
			Service: req.Service,
			Count:   deleted,
		})
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

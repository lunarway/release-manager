package http

import (
	"encoding/json"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
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
			addAutoReleasePolicy(configRepo, sshPrivateKeyPath)(w, r)
			return
		default:
			Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}

func addAutoReleasePolicy(configRepo, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.AddAutoReleasePolicyRequest
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

		log.Infof("http add auto-release policy started: service '%s' branch '%s' environment '%s'", req.Service, req.Branch, req.Environment)
		id, err := policyinternal.AddAutoRelease(r.Context(), configRepo, sshPrivateKeyPath, req.Service, req.Branch, req.Environment, req.CommitterName, req.CommitterEmail)
		if err != nil {
			log.Errorf("http add auto-release policy failed: config repo '%s' service '%s' branch '%s' environment '%s': %v", configRepo, req.Service, req.Branch, req.Environment, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(httpinternal.AddPolicyResponse{
			ID:          id,
			Service:     req.Service,
			Branch:      req.Branch,
			Environment: req.Environment,
		})
		if err != nil {
			log.Errorf("http add auto-release policy failed: config repo '%s' service '%s' branch '%s' environment '%s': encode response: %v", configRepo, req.Service, req.Branch, req.Environment, err)
			Error(w, "unknown error", http.StatusInternalServerError)
			return
		}
	}
}

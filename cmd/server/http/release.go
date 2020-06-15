package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func release(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.ReleaseRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: release: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		if !req.Validate(w) {
			return
		}
		logger = logger.WithFields(
			"service", req.Service,
			"req", req)
		var releaseID string
		switch {
		case !emptyString(req.Branch):
			logger.Infof("http: release: service '%s' environment '%s' branch '%s': releasing branch", req.Service, req.Environment, req.Branch)
			releaseID, err = flowSvc.ReleaseBranch(ctx, flow.Actor{
				Name:  req.CommitterName,
				Email: req.CommitterEmail,
			}, req.Environment, req.Service, req.Branch)
		case !emptyString(req.ArtifactID):
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': releasing artifact", req.Service, req.Environment, req.ArtifactID)
			releaseID, err = flowSvc.ReleaseArtifactID(ctx, flow.Actor{
				Name:  req.CommitterName,
				Email: req.CommitterEmail,
			}, req.Environment, req.Service, req.ArtifactID)
		default:
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s' branch '%s': neither branch nor artifact id specified", req.Service, req.Environment, req.ArtifactID, req.Branch)
			httpinternal.Error(w, "either branch or artifact id must be specified", http.StatusBadRequest)
			return
		}
		var statusString string
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release cancelled", req.Service, req.Environment, req.Branch, req.ArtifactID)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case flow.ErrReleaseProhibited:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: branch prohibited in environment: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				if req.Branch != "" {
					httpinternal.Error(w, fmt.Sprintf("branch '%s' cannot be released to environment '%s' due to branch restriction policy", req.Branch, req.Environment), http.StatusBadRequest)
				} else {
					httpinternal.Error(w, fmt.Sprintf("artifact '%s' cannot be released to environment '%s' due to branch restriction policy", req.ArtifactID, req.Environment), http.StatusBadRequest)
				}
				return
			case flow.ErrNothingToRelease:
				statusString = "Environment is already up-to-date"
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release skipped: environment up to date: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
			case git.ErrArtifactNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				httpinternal.Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				httpinternal.Error(w, "could not release right now. Please try again in a moment.", http.StatusServiceUnavailable)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				if req.Branch != "" {
					httpinternal.Error(w, fmt.Sprintf("artifact for branch '%s' not found for service '%s'", req.Branch, req.Service), http.StatusBadRequest)
				} else {
					httpinternal.Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				}
				return
			case flow.ErrUnknownEnvironment:
				logger.Infof("http: release: service '%s' environment '%s': release rejected: %v", req.Service, req.Environment, err)
				httpinternal.Error(w, fmt.Sprintf("unknown environment: %s", req.Environment), http.StatusBadRequest)
				return
			case flow.ErrUnknownConfiguration:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: source configuration not found: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				httpinternal.Error(w, fmt.Sprintf("configuration for environment '%s' not found for service '%s'. Is the environment specified in 'shuttle.yaml'?", req.Environment, req.Service), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release failed: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.ReleaseResponse{
			Service:       req.Service,
			ReleaseID:     releaseID,
			ToEnvironment: req.Environment,
			Tag:           releaseID,
			Status:        statusString,
		})
		if err != nil {
			logger.Errorf("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': marshal response failed: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
		}
	}
}

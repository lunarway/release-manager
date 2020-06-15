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

func rollback(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpinternal.Error(w, "not found", http.StatusNotFound)
			return
		}
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.RollbackRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: rollback failed: decode request body: %v", err)
			httpinternal.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		// default namespace to environment if it's empty. For most devlopers this
		// allows them to avoid setting the namespace flag for requests.
		if emptyString(req.Namespace) {
			req.Namespace = req.Environment
		}
		if !req.Validate(w) {
			return
		}

		logger = logger.WithFields("service", req.Service, "namespace", req.Namespace, "req", req)
		res, err := flowSvc.Rollback(ctx, flow.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Environment, req.Namespace, req.Service)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: rollback cancelled: env '%s' service '%s'", req.Environment, req.Service)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case flow.ErrNamespaceNotAllowedByArtifact:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				httpinternal.Error(w, "namespace not allowed by artifact", http.StatusBadRequest)
				return
			case git.ErrReleaseNotFound:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				httpinternal.Error(w, fmt.Sprintf("no release of service '%s' available for rollback in environment '%s'", req.Service, req.Environment), http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: rollback: service '%s' environment '%s': %v", req.Service, req.Environment, err)
				httpinternal.Error(w, "could not roll back right now. Please try again in a moment.", http.StatusServiceUnavailable)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				httpinternal.Error(w, fmt.Sprintf("no release of service '%s' available for rollback in environment '%s'. Are you missing a namespace?", req.Service, req.Environment), http.StatusBadRequest)
				return
			case git.ErrNothingToCommit:
				logger.Infof("http: rollback rejected: env '%s' service '%s': already rolled back: %v", req.Environment, req.Service, err)
				httpinternal.Error(w, fmt.Sprintf("service '%s' already rolled back in environment '%s'", req.Service, req.Environment), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: rollback failed: env '%s' service '%s': %v", req.Environment, req.Service, err)
				httpinternal.Error(w, "unknown error", http.StatusInternalServerError)
				return
			}
		}
		var status string
		if res.OverwritingNamespace != "" {
			status = fmt.Sprintf("Namespace '%s' did not match that of the artifact and was overwritten to '%s'", req.Namespace, res.OverwritingNamespace)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = payload.encodeResponse(ctx, w, httpinternal.RollbackResponse{
			Status:             status,
			Service:            req.Service,
			Environment:        req.Environment,
			PreviousArtifactID: res.Previous,
			NewArtifactID:      res.New,
		})
		if err != nil {
			logger.Errorf("http: rollback failed: env '%s' service '%s': marshal response: %v", req.Environment, req.Service, err)
		}
	}
}

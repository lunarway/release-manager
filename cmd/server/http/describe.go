package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func muxService(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["service"]
}

func muxEnvironment(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["environment"]
}

func describeRelease(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := muxService(r)
		environment := muxEnvironment(r)

		values := r.URL.Query()
		namespace := values.Get("namespace")
		countParam := values.Get("count")

		if emptyString(countParam) {
			countParam = "1"
		}
		count, err := strconv.Atoi(countParam)
		if err != nil || count <= 0 {
			httpinternal.Error(w, fmt.Sprintf("invalid value '%s' of count. Must be a positive integer.", countParam), http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		logger := log.WithContext(ctx).WithFields("service", service, "environment", environment, "namespace", namespace)
		resp, err := flowSvc.DescribeRelease(ctx, environment, service, count)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: describe release: service '%s' environment '%s': request cancelled", service, environment)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case artifact.ErrFileNotFound:
				httpinternal.Error(w, fmt.Sprintf("no release of service '%s' available in environment '%s'. Are you missing a namespace?", service, environment), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: describe release: service '%s' environment '%s': failed: %v", service, environment, err)
				unknownError(w)
				return
			}
		}

		var releases []httpinternal.DescribeReleaseResponseRelease
		for _, release := range resp.Releases {
			releases = append(releases, httpinternal.DescribeReleaseResponseRelease{
				ReleaseIndex:    release.ReleaseIndex,
				Artifact:        release.Artifact,
				ReleasedAt:      release.ReleasedAt,
				ReleasedByName:  release.ReleasedByName,
				ReleasedByEmail: release.ReleasedByEmail,
				Intent:          release.Intent,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DescribeReleaseResponse{
			Service:     service,
			Environment: environment,
			Releases:    releases,
		})
		if err != nil {
			logger.Errorf("http: describe release: service '%s' environment '%s': marshal response failed: %v", service, environment, err)
		}
	}
}

func describeArtifact(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := muxService(r)
		values := r.URL.Query()
		countParam := values.Get("count")
		if emptyString(countParam) {
			countParam = "1"
		}
		count, err := strconv.Atoi(countParam)
		if err != nil || count <= 0 {
			httpinternal.Error(w, fmt.Sprintf("invalid value '%s' of count. Must be a positive integer.", countParam), http.StatusBadRequest)
			return
		}
		branch := values.Get("branch")
		ctx := r.Context()
		logger := log.WithContext(ctx).WithFields("service", service, "count", count, "branch", branch)
		resp, err := flowSvc.DescribeArtifact(ctx, service, count, branch)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: describe artifact: service '%s': request cancelled", service)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case flow.ErrArtifactNotFound:
				httpinternal.Error(w, fmt.Sprintf("no artifacts available for service '%s'.", service), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: describe artifact: service '%s': failed: %v", service, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DescribeArtifactResponse{
			Service:   service,
			Artifacts: resp,
		})
		if err != nil {
			logger.Errorf("http: describe artifact: service '%s': marshal response failed: %v", service, err)
		}
	}
}

func describeLatestArtifacts(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := muxService(r)
		values := r.URL.Query()
		branch := values.Get("branch")
		if emptyString(branch) {
			requiredFieldError(w, "branch")
			return
		}

		ctx := r.Context()
		logger := log.WithContext(ctx).WithFields("service", service, "branch", branch)
		resp, err := flowSvc.DescribeLatestArtifact(ctx, service, branch)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: describe latest artifact: service '%s': request cancelled", service)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case flow.ErrArtifactNotFound:
				httpinternal.Error(w, fmt.Sprintf("no artifacts available for service '%s' and branch '%s'.", service, branch), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: describe latest artifact: service '%s' and branch '%s': failed: %v", service, branch, err)
				unknownError(w)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DescribeArtifactResponse{
			Service:   service,
			Artifacts: []artifact.Spec{resp},
		})
		if err != nil {
			logger.Errorf("http: describe latest artifact: service '%s' and branch '%s': marshal response failed: %v", service, branch, err)
		}
	}
}

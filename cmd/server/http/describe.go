package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func describe(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			notFound(w)
			return
		}
		p, ok := newDescribePath(r)
		if !ok {
			notFound(w)
			return
		}
		ctx := r.Context()
		switch p.Resource() {
		case "release":
			describeRelease(ctx, payload, flowSvc, p.Namespace(), p.Environment(), p.Service())(w, r)
		case "artifact":
			describeArtifact(ctx, payload, flowSvc, p.Service())(w, r)
		default:
			log.WithContext(ctx).Errorf("describe path not found: %+v", p)
			notFound(w)
		}
	}
}

type describePath struct {
	r        *http.Request
	segments []string
}

func newDescribePath(r *http.Request) (describePath, bool) {
	p := describePath{
		r:        r,
		segments: strings.Split(r.URL.Path, "/"),
	}
	if len(p.segments) < 4 {
		return describePath{}, false
	}
	return p, true
}

func (p *describePath) Resource() string {
	return p.segments[2]
}

func (p *describePath) Service() string {
	return p.segments[3]
}

func (p *describePath) Environment() string {
	if len(p.segments) < 5 {
		return ""
	}
	return p.segments[4]
}

func (p *describePath) Namespace() string {
	values := p.r.URL.Query()
	namespace := values.Get("namespace")
	return namespace
}

func describeRelease(ctx context.Context, payload *payload, flowSvc *flow.Service, namespace, environment, service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if emptyString(service) {
			requiredFieldError(w, "service")
			return
		}
		if emptyString(environment) {
			requiredFieldError(w, "environment")
			return
		}
		logger := log.WithContext(ctx).WithFields("service", service, "environment", environment, "namespace", namespace)
		ctx := r.Context()
		resp, err := flowSvc.DescribeRelease(ctx, namespace, environment, service)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.DescribeReleaseResponse{
			Service:         service,
			Environment:     environment,
			Artifact:        resp.Artifact,
			ReleasedAt:      resp.ReleasedAt,
			ReleasedByEmail: resp.ReleasedByEmail,
			ReleasedByName:  resp.ReleasedByName,
		})
		if err != nil {
			logger.Errorf("http: describe release: service '%s' environment '%s': marshal response failed: %v", service, environment, err)
		}
	}
}

func describeArtifact(ctx context.Context, payload *payload, flowSvc *flow.Service, service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if emptyString(service) {
			requiredFieldError(w, "service")
			return
		}
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
		logger := log.WithContext(ctx).WithFields("service", service, "count", count)
		ctx := r.Context()
		resp, err := flowSvc.DescribeArtifact(ctx, service, count)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: describe artifact: service '%s': request cancelled", service)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case git.ErrArtifactNotFound:
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

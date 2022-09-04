package http

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Options struct {
	Port                int
	Timeout             time.Duration
	GithubWebhookSecret string
	HamCtlAuthToken     string
	DaemonAuthToken     string
	ArtifactAuthToken   string
	S3WebhookSecret     string
}

func NewServer(opts *Options, slackClient *slack.Client, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, artifactWriteStorage ArtifactWriteStorage, logger *log.Logger, tracer tracing.Tracer) error {
	payloader := payload{
		tracer: tracer,
	}
	m := mux.NewRouter()
	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.WithContext(r.Context()).Infof("Unknown HTTP endpoint called: %s", r.URL.String())
		notFound(w, logger)
	})

	m.Use(trace(tracer))
	m.Use(prometheusMiddleware())
	m.Use(reqrespLogger(logger))

	hamctlMux := m.NewRoute().Subrouter()
	hamctlMux.Use(authenticate(logger, opts.HamCtlAuthToken))
	hamctlMux.Methods(http.MethodPost).Path("/release").Handler(release(&payloader, flowSvc, logger))
	hamctlMux.Methods(http.MethodGet).Path("/status").Handler(status(&payloader, flowSvc, logger))

	policyMux := hamctlMux.PathPrefix("/policies").Subrouter()
	policyMux.Methods(http.MethodGet).Handler(listPolicies(&payloader, policySvc, logger))
	policyMux.Methods(http.MethodDelete).Handler(deletePolicies(&payloader, policySvc, logger))
	policyMux.Methods(http.MethodPatch).Path("/auto-release").Handler(applyAutoReleasePolicy(&payloader, policySvc, logger))
	policyMux.Methods(http.MethodPatch).Path("/branch-restriction").Handler(applyBranchRestrictionPolicy(&payloader, policySvc, logger))

	hamctlMux.Methods(http.MethodGet).Path("/describe/release/{service}/{environment}").Handler(describeRelease(&payloader, flowSvc, logger))
	hamctlMux.Methods(http.MethodGet).Path("/describe/artifact/{service}").Handler(describeArtifact(&payloader, flowSvc, logger))
	hamctlMux.Methods(http.MethodGet).Path("/describe/latest-artifact/{service}").Handler(describeLatestArtifacts(&payloader, flowSvc, logger))

	daemonMux := m.NewRoute().Subrouter()
	daemonMux.Use(authenticate(logger, opts.DaemonAuthToken))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/deploy").Handler(daemonk8sDeployWebhook(&payloader, flowSvc, logger))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/error").Handler(daemonk8sPodErrorWebhook(&payloader, flowSvc, logger))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/joberror").Handler(daemonk8sJobErrorWebhook(&payloader, flowSvc, logger))

	// s3 endpoints
	artifactMux := m.NewRoute().Subrouter()
	artifactMux.Use(authenticate(logger, opts.ArtifactAuthToken))
	artifactMux.Methods(http.MethodPost).Path("/artifacts/create").Handler(createArtifact(&payloader, artifactWriteStorage, logger))

	// profiling endpoints
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)

	m.HandleFunc("/ping", ping)
	m.Handle("/metrics", promhttp.Handler())
	m.HandleFunc("/webhook/github", githubWebhook(&payloader, flowSvc, policySvc, gitSvc, slackClient, logger, opts.GithubWebhookSecret))

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", opts.Port),
		Handler:           m,
		ReadTimeout:       opts.Timeout,
		WriteTimeout:      opts.Timeout,
		IdleTimeout:       opts.Timeout,
		ReadHeaderTimeout: opts.Timeout,
	}
	logger.Infof("Initializing HTTP Server on port %d", opts.Port)
	err := s.ListenAndServe()
	if err != nil {
		return errors.WithMessage(err, "listen and server")
	}
	return nil
}

// getRequestID returns the request ID of an HTTP request if it is set and
// otherwise generates a new one.
func getRequestID(r *http.Request) string {
	requestID := r.Header.Get("x-request-id")
	if requestID != "" {
		return requestID
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return ""
	}
	r.Header.Set("x-request-id", requestID)
	return id.String()
}

// trace adds an OpenTracing span to the request context.
func trace(tracer tracing.Tracer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			span, ctx := tracer.FromCtxf(ctx, "http %s %s", r.Method, r.URL.Path)
			defer span.Finish()
			requestID := getRequestID(r)
			ctx = tracing.WithRequestID(ctx, requestID)
			ctx = log.AddContext(ctx, "requestId", requestID)
			*r = *r.WithContext(ctx)
			statusWriter := &statusCodeResponseWriter{w, http.StatusOK}
			h.ServeHTTP(statusWriter, r)
			span.SetTag("request.id", requestID)
			span.SetTag("http.status_code", statusWriter.statusCode)
			span.SetTag("http.url", r.URL.RequestURI())
			span.SetTag("http.method", r.Method)
			if statusWriter.statusCode >= http.StatusInternalServerError {
				span.SetTag("error", true)
			}
			err := ctx.Err()
			if err != nil {
				span.SetTag("error", true)
				span.SetTag("error_message", err.Error())
			}
		})
	}
}

// authenticate authenticates the handler against a Bearer token.
//
// If authentication fails a 401 Unauthorized HTTP status is returned with an
// ErrorResponse body.
func authenticate(logger *log.Logger, token string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			t := strings.TrimPrefix(authorization, "Bearer ")
			t = strings.TrimSpace(t)
			if t != token {
				httpinternal.Error(w, logger, "please provide a valid authentication token", http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

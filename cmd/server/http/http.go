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

func NewServer(opts *Options, slackClient *slack.Client, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, artifactWriteStorage ArtifactWriteStorage, tracer tracing.Tracer) error {
	payloader := payload{
		tracer: tracer,
	}
	m := mux.NewRouter()
	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithContext(r.Context()).Infof("Unknown HTTP endpoint called: %s", r.URL.String())
		notFound(w)
	})

	m.Use(trace(tracer))
	m.Use(prometheusMiddleware())
	m.Use(reqrespLogger)

	hamctlMux := m.NewRoute().Subrouter()
	hamctlMux.Use(authenticate(opts.HamCtlAuthToken))
	hamctlMux.Methods(http.MethodPost).Path("/release").Handler(release(&payloader, flowSvc))
	hamctlMux.Methods(http.MethodGet).Path("/status").Handler(status(&payloader, flowSvc))

	policyMux := hamctlMux.PathPrefix("/policies").Subrouter()
	policyMux.Methods(http.MethodGet).Handler(listPolicies(&payloader, policySvc))
	policyMux.Methods(http.MethodDelete).Handler(deletePolicies(&payloader, policySvc))
	policyMux.Methods(http.MethodPatch).Path("/auto-release").Handler(applyAutoReleasePolicy(&payloader, policySvc))
	policyMux.Methods(http.MethodPatch).Path("/branch-restriction").Handler(applyBranchRestrictionPolicy(&payloader, policySvc))

	hamctlMux.Methods(http.MethodGet).Path("/describe/release/{service}/{environment}").Handler(describeRelease(&payloader, flowSvc))
	hamctlMux.Methods(http.MethodGet).Path("/describe/artifact/{service}").Handler(describeArtifact(&payloader, flowSvc))
	hamctlMux.Methods(http.MethodGet).Path("/describe/latest-artifact/{service}").Handler(describeLatestArtifacts(&payloader, flowSvc))

	daemonMux := m.NewRoute().Subrouter()
	daemonMux.Use(authenticate(opts.DaemonAuthToken))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/deploy").Handler(daemonk8sDeployWebhook(&payloader, flowSvc))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/error").Handler(daemonk8sPodErrorWebhook(&payloader, flowSvc))
	daemonMux.Methods(http.MethodPost).Path("/webhook/daemon/k8s/joberror").Handler(daemonk8sJobErrorWebhook(&payloader, flowSvc))

	// s3 endpoints
	artifactMux := m.NewRoute().Subrouter()
	artifactMux.Use(authenticate(opts.ArtifactAuthToken))
	artifactMux.Methods(http.MethodPost).Path("/artifacts/create").Handler(createArtifact(&payloader, artifactWriteStorage))

	// profiling endpoints
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)

	m.HandleFunc("/ping", ping)
	m.Handle("/metrics", promhttp.Handler())
	m.HandleFunc("/webhook/github", githubWebhook(&payloader, flowSvc, policySvc, gitSvc, slackClient, opts.GithubWebhookSecret))

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", opts.Port),
		Handler:           m,
		ReadTimeout:       opts.Timeout,
		WriteTimeout:      opts.Timeout,
		IdleTimeout:       opts.Timeout,
		ReadHeaderTimeout: opts.Timeout,
	}
	log.Infof("Initializing HTTP Server on port %d", opts.Port)
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
func authenticate(token string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("X-HAM-TOKEN")
			t := strings.TrimSpace(authorization)
			if t != token {
				httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

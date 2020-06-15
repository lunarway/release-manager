package http

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/google/uuid"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", trace(tracer, authenticate(opts.HamCtlAuthToken, promote(&payloader, flowSvc))))
	mux.HandleFunc("/release", trace(tracer, authenticate(opts.HamCtlAuthToken, release(&payloader, flowSvc))))
	mux.HandleFunc("/status", trace(tracer, authenticate(opts.HamCtlAuthToken, status(&payloader, flowSvc))))
	mux.HandleFunc("/rollback", trace(tracer, authenticate(opts.HamCtlAuthToken, rollback(&payloader, flowSvc))))
	// register both a rooted and unrooted path to avoid a 301 redirect on /policies when only /policies/ is registered
	mux.HandleFunc("/policies", trace(tracer, authenticate(opts.HamCtlAuthToken, policy(&payloader, policySvc))))
	mux.HandleFunc("/policies/", trace(tracer, authenticate(opts.HamCtlAuthToken, policy(&payloader, policySvc))))
	mux.HandleFunc("/describe/", trace(tracer, authenticate(opts.HamCtlAuthToken, describe(&payloader, flowSvc))))
	mux.HandleFunc("/webhook/github", trace(tracer, githubWebhook(&payloader, flowSvc, policySvc, gitSvc, slackClient, opts.GithubWebhookSecret)))
	mux.HandleFunc("/webhook/daemon/flux", trace(tracer, authenticate(opts.DaemonAuthToken, daemonFluxWebhook(&payloader, flowSvc))))
	mux.HandleFunc("/webhook/daemon/k8s/deploy", trace(tracer, authenticate(opts.DaemonAuthToken, daemonk8sDeployWebhook(&payloader, flowSvc))))
	mux.HandleFunc("/webhook/daemon/k8s/error", trace(tracer, authenticate(opts.DaemonAuthToken, daemonk8sPodErrorWebhook(&payloader, flowSvc))))

	// s3 endpoints
	mux.HandleFunc("/artifacts/create", trace(tracer, authenticate(opts.ArtifactAuthToken, createArtifact(&payloader, artifactWriteStorage))))

	// profiling endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.Handle("/metrics", promhttp.Handler())

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", opts.Port),
		Handler:           reqrespLogger(mux),
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
func trace(tracer tracing.Tracer, h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span, ctx := tracer.FromCtxf(ctx, "http %s %s", r.Method, r.URL.Path)
		defer span.Finish()
		requestID := getRequestID(r)
		ctx = tracing.WithRequestID(ctx, requestID)
		ctx = log.AddContext(ctx, "requestId", requestID)
		*r = *r.WithContext(ctx)
		statusWriter := &statusCodeResponseWriter{w, http.StatusOK}
		h(statusWriter, r)
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

// authenticate authenticates the handler against a Bearer token.
//
// If authentication fails a 401 Unauthorized HTTP status is returned with an
// ErrorResponse body.
func authenticate(token string, h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		t := strings.TrimPrefix(authorization, "Bearer ")
		t = strings.TrimSpace(t)
		if t != token {
			httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
			return
		}
		h(w, r)
	})
}

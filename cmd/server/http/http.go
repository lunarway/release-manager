package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/multierr"
	"gopkg.in/go-playground/webhooks.v5/github"
)

type Options struct {
	Port                int
	Timeout             time.Duration
	GithubWebhookSecret string
	HamCtlAuthToken     string
	DaemonAuthToken     string
}

func NewServer(opts *Options, slackClient *slack.Client, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, tracer tracing.Tracer) error {
	payloader := payload{
		tracer: tracer,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", trace(tracer, authenticate(opts.HamCtlAuthToken, promote(&payloader, flowSvc))))
	mux.HandleFunc("/release", trace(tracer, authenticate(opts.HamCtlAuthToken, release(&payloader, flowSvc))))
	mux.HandleFunc("/status", trace(tracer, authenticate(opts.HamCtlAuthToken, status(&payloader, flowSvc))))
	mux.HandleFunc("/rollback", trace(tracer, authenticate(opts.HamCtlAuthToken, rollback(&payloader, flowSvc))))
	mux.HandleFunc("/policies", trace(tracer, authenticate(opts.HamCtlAuthToken, policy(&payloader, policySvc))))
	mux.HandleFunc("/describe/", trace(tracer, authenticate(opts.HamCtlAuthToken, describe(&payloader, flowSvc))))
	mux.HandleFunc("/webhook/github", trace(tracer, githubWebhook(&payloader, flowSvc, policySvc, gitSvc, slackClient, opts.GithubWebhookSecret)))
	mux.HandleFunc("/webhook/daemon", trace(tracer, authenticate(opts.DaemonAuthToken, daemonWebhook(&payloader, flowSvc))))

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
			Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
			return
		}
		h(w, r)
	})
}

// payload is a struct tracing encoding and deconding operations of HTTP payloads.
type payload struct {
	tracer tracing.Tracer
}

// encodeResponse encodes resp as JSON into w. Tracing is reported from the
// context ctx and reported on tracer.
func (p *payload) encodeResponse(ctx context.Context, w io.Writer, resp interface{}) error {
	span, _ := p.tracer.FromCtx(ctx, "json encode response")
	defer span.Finish()
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		return err
	}
	return nil
}

// decodeResponse decodes req as JSON into r. Tracing is reported from the
// context ctx and reported on tracer.
func (p *payload) decodeResponse(ctx context.Context, r io.Reader, req interface{}) error {
	span, _ := p.tracer.FromCtx(ctx, "json decode request")
	defer span.Finish()
	decoder := json.NewDecoder(r)
	err := decoder.Decode(req)
	if err != nil {
		return err
	}
	return nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func status(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		namespace := values.Get("namespace")
		service := values.Get("service")
		if emptyString(service) {
			requiredQueryError(w, "service")
			return
		}

		ctx := r.Context()
		logger := log.WithContext(ctx).WithFields("service", service, "namespace", namespace)
		s, err := flowSvc.Status(ctx, namespace, service)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: status: get status cancelled: service '%s'", service)
				cancelled(w)
				return
			}
			logger.Errorf("http: status: get status failed: service '%s': %v", service, err)
			unknownError(w)
			return
		}

		dev := httpinternal.Environment{
			Message:               s.Dev.Message,
			Author:                s.Dev.Author,
			Tag:                   s.Dev.Tag,
			Committer:             s.Dev.Committer,
			Date:                  convertTimeToEpoch(s.Dev.Date),
			BuildUrl:              s.Dev.BuildURL,
			HighVulnerabilities:   s.Dev.HighVulnerabilities,
			MediumVulnerabilities: s.Dev.MediumVulnerabilities,
			LowVulnerabilities:    s.Dev.LowVulnerabilities,
		}

		staging := httpinternal.Environment{
			Message:               s.Staging.Message,
			Author:                s.Staging.Author,
			Tag:                   s.Staging.Tag,
			Committer:             s.Staging.Committer,
			Date:                  convertTimeToEpoch(s.Staging.Date),
			BuildUrl:              s.Staging.BuildURL,
			HighVulnerabilities:   s.Staging.HighVulnerabilities,
			MediumVulnerabilities: s.Staging.MediumVulnerabilities,
			LowVulnerabilities:    s.Staging.LowVulnerabilities,
		}

		prod := httpinternal.Environment{
			Message:               s.Prod.Message,
			Author:                s.Prod.Author,
			Tag:                   s.Prod.Tag,
			Committer:             s.Prod.Committer,
			Date:                  convertTimeToEpoch(s.Prod.Date),
			BuildUrl:              s.Prod.BuildURL,
			HighVulnerabilities:   s.Prod.HighVulnerabilities,
			MediumVulnerabilities: s.Prod.MediumVulnerabilities,
			LowVulnerabilities:    s.Prod.LowVulnerabilities,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = payload.encodeResponse(ctx, w, httpinternal.StatusResponse{
			DefaultNamespaces: s.DefaultNamespaces,
			Dev:               &dev,
			Staging:           &staging,
			Prod:              &prod,
		})
		if err != nil {
			logger.Errorf("http: status: service '%s': marshal response failed: %v", service, err)
		}
	}
}

func rollback(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			Error(w, "not found", http.StatusNotFound)
			return
		}
		ctx := r.Context()
		logger := log.WithContext(ctx)
		var req httpinternal.RollbackRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: rollback failed: decode request body: %v", err)
			Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if emptyString(req.Service) {
			requiredFieldError(w, "service")
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
		// default namespace to environment if it's empty. For most devlopers this
		// allows them to avoid setting the namespace flag for requests.
		if emptyString(req.Namespace) {
			req.Namespace = req.Environment
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
				Error(w, "namespace not allowed by artifact", http.StatusBadRequest)
				return
			case git.ErrReleaseNotFound:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				Error(w, fmt.Sprintf("no release of service '%s' available for rollback in environment '%s'", req.Service, req.Environment), http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: rollback: service '%s' environment '%s': %v", req.Service, req.Environment, err)
				Error(w, fmt.Sprintf("could not roll back right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				Error(w, fmt.Sprintf("no release of service '%s' available for rollback in environment '%s'. Are you missing a namespace?", req.Service, req.Environment), http.StatusBadRequest)
				return
			case git.ErrNothingToCommit:
				logger.Infof("http: rollback rejected: env '%s' service '%s': already rolled back", req.Environment, req.Service)
				Error(w, fmt.Sprintf("service '%s' already rolled back in environment '%s'", req.Service, req.Environment), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: rollback failed: env '%s' service '%s': %v", req.Environment, req.Service, err)
				Error(w, "unknown error", http.StatusInternalServerError)
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

func daemonWebhook(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		var podNotify httpinternal.PodNotifyRequest
		err := payload.decodeResponse(ctx, r.Body, &podNotify)
		if err != nil {
			logger.Errorf("http: daemon webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		logger = logger.WithFields("pod", podNotify.Name,
			"namespace", podNotify.Namespace,
			"environment", podNotify.Environment,
			"state", podNotify.State,
			"message", podNotify.Message,
			"reason", podNotify.Reason,
			"artifactId", podNotify.ArtifactID,
			"raw", podNotify)

		err = flowSvc.NotifyCommitter(ctx, &podNotify)
		if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
			logger.Errorf("http: daemon webhook failed: pod '%s' namespace '%s' environment: '%s' notify committer: %v", podNotify.Name, podNotify.Namespace, podNotify.Environment, err)
			unknownError(w)
			return
		}

		logger.Infof("http: daemon webhook: pod '%s' namespace '%s' environment '%s': Pod event handled: state=%s", podNotify.Name, podNotify.Namespace, podNotify.Environment, podNotify.State)
	}
}

func githubWebhook(payload *payload, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, slackClient *slack.Client, githubWebhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		hook, _ := github.New(github.Options.Secret(githubWebhookSecret))
		payload, err := hook.Parse(r, github.PushEvent)
		if err != nil {
			logger.Errorf("http: github webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		switch payload := payload.(type) {
		case github.PushPayload:
			if !isBranchPush(payload.Ref) {
				logger.Infof("http: github webhook: ref '%s' is not a branch push", payload.Ref)
				w.WriteHeader(http.StatusOK)
				return
			}
			err := gitSvc.SyncMaster(ctx)
			if err != nil {
				logger.Errorf("http: github webhook: failed to sync master: %v", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			rgx := regexp.MustCompile(`\[(.*?)\]`)
			matches := rgx.FindStringSubmatch(payload.HeadCommit.Message)
			if len(matches) < 2 {
				logger.Infof("http: github webhook: no service match from commit '%s'", payload.HeadCommit.Message)
				w.WriteHeader(http.StatusOK)
				return
			}
			serviceName := matches[1]

			// locate branch of commit
			branch, ok := git.BranchName(payload.HeadCommit.Modified, flowSvc.ArtifactFileName, serviceName)
			if !ok {
				logger.Infof("http: github webhook: service '%s': branch name not found", serviceName)
				w.WriteHeader(http.StatusOK)
				return
			}

			logger = logger.WithFields("branch", branch, "service", serviceName, "commit", payload.HeadCommit)
			// lookup policies for branch
			autoReleases, err := policySvc.GetAutoReleases(ctx, serviceName, branch)
			if err != nil {
				logger.Errorf("http: github webhook: service '%s' branch '%s': get auto release policies failed: %v", serviceName, branch, err)
				err := slackClient.NotifySlackPolicyFailed(ctx, payload.HeadCommit.Author.Email, "Auto release policy failed", fmt.Sprintf("failed for branch: %s, env: %s", serviceName, branch))
				if err != nil {
					logger.Errorf("http: github webhook: get auto-release policies: error notifying slack: %v", err)
				}
				unknownError(w)
				return
			}
			logger.Infof("http: github webhook: service '%s' branch '%s': found %d release policies", serviceName, branch, len(autoReleases))
			var errs error
			for _, autoRelease := range autoReleases {
				releaseID, err := flowSvc.ReleaseBranch(ctx, flow.Actor{
					Name:  payload.HeadCommit.Author.Name,
					Email: payload.HeadCommit.Author.Email,
				}, autoRelease.Environment, serviceName, autoRelease.Branch)
				if err != nil {
					if errorCause(err) != git.ErrNothingToCommit {
						errs = multierr.Append(errs, err)
						err := slackClient.NotifySlackPolicyFailed(ctx, payload.HeadCommit.Author.Email, "Auto release policy failed", fmt.Sprintf("Service %s was not released into %s from branch %s.\nYou can deploy manually using `hamctl`:\nhamctl release --service %[1]s --branch %[3]s --env %[2]s", serviceName, autoRelease.Environment, autoRelease.Branch))
						if err != nil {
							logger.Errorf("http: github webhook: auto-release failed: error notifying slack: %v", err)
						}
						continue
					}
					logger.Infof("http: github webhook: service '%s': auto-release from policy '%s' to '%s': nothing to commit", serviceName, autoRelease.ID, autoRelease.Environment)
					continue
				}
				err = slackClient.NotifySlackPolicySucceeded(ctx, payload.HeadCommit.Author.Email, "Auto release policy detected", fmt.Sprintf("Service *%s* was auto released to *%s*", serviceName, autoRelease.Environment))
				if err != nil {
					if errors.Cause(err) != slack.ErrUnknownEmail {
						logger.Errorf("http: github webhook: auto-release succeeded: error notifying slack: %v", err)
					}
				}
				logger.Infof("http: github webhook: service '%s': auto-release from policy '%s' of %s to %s", serviceName, autoRelease.ID, releaseID, autoRelease.Environment)
			}
			if errs != nil {
				logger.Errorf("http: github webhook: service '%s' branch '%s': auto-release failed with one or more errors: %v", serviceName, branch, errs)
				unknownError(w)
				return
			}
			logger.Infof("http: github webhook: handled successfully: service '%s' branch '%s' commit '%s'", serviceName, branch, payload.HeadCommit.ID)
			w.WriteHeader(http.StatusOK)
			return
		default:
			logger.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}

func isBranchPush(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/")
}

func promote(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req httpinternal.PromoteRequest
		ctx := r.Context()
		logger := log.WithContext(ctx)
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: promote: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		// default namespace to environment if it's empty. For most devlopers this
		// allows them to avoid setting the namespace flag for requests.
		if emptyString(req.Namespace) {
			req.Namespace = req.Environment
		}

		logger = logger.WithFields("service", req.Service, "namespace", req.Namespace, "req", req)
		result, err := flowSvc.Promote(ctx, flow.Actor{
			Name:  req.CommitterName,
			Email: req.CommitterEmail,
		}, req.Environment, req.Namespace, req.Service)

		var statusString string
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: promote: service '%s' environment '%s': promote cancelled", req.Service, req.Environment)
				cancelled(w)
				return
			}
			switch errorCause(err) {
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
				logger.Infof("http: promote: service '%s' environment '%s': promote skipped: environment up to date", req.Service, req.Environment)
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: promote: service '%s' environment '%s': %v", req.Service, req.Environment, err)
				Error(w, fmt.Sprintf("could not promote right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			case flow.ErrUnknownEnvironment:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: %v", req.Service, req.Environment, err)
				Error(w, fmt.Sprintf("unknown environment: %s", req.Environment), http.StatusBadRequest)
				return
			case flow.ErrNamespaceNotAllowedByArtifact:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: %v", req.Service, req.Environment, err)
				Error(w, "namespace not allowed by artifact", http.StatusBadRequest)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: artifact not found", req.Service, req.Environment)
				Error(w, fmt.Sprintf("artifact not found for service '%s'. Are you missing a namespace?", req.Service), http.StatusBadRequest)
				return
			case flow.ErrUnknownConfiguration:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: source configuration not found", req.Service, req.Environment)
				Error(w, fmt.Sprintf("configuration for environment '%s' not found for service '%s'. Is the environment specified in 'shuttle.yaml'?", req.Environment, req.Service), http.StatusBadRequest)
				return
			default:
				logger.Errorf("http: promote: service '%s' environment '%s': promote failed: %v", req.Service, req.Environment, err)
				unknownError(w)
				return
			}
		}

		var fromEnvironment string
		switch req.Environment {
		case "dev":
			fromEnvironment = "master"
		case "staging":
			fromEnvironment = "dev"
		case "prod":
			fromEnvironment = "staging"
		default:
			fromEnvironment = req.Environment
		}

		if result.OverwritingNamespace != "" {
			statusString = fmt.Sprintf("Namespace '%s' did not match that of the artifact and was overwritten to '%s'", req.Namespace, result.OverwritingNamespace)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = payload.encodeResponse(ctx, w, httpinternal.PromoteResponse{
			Service:         req.Service,
			FromEnvironment: fromEnvironment,
			ToEnvironment:   req.Environment,
			Tag:             result.ReleaseID,
			Status:          statusString,
		})
		if err != nil {
			logger.Errorf("http: promote: service '%s' environment '%s': marshal response failed: %v", req.Service, req.Environment, err)
		}
	}
}

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
		logger = logger.WithFields(
			"service", req.Service,
			"req", req)
		var releaseID string
		switch {
		case req.Branch != "" && req.ArtifactID != "":
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s' branch '%s': brand and artifact id both specified", req.Service, req.Environment, req.ArtifactID, req.Branch)
			Error(w, "branch and artifact id cannot both be specified. Pick one", http.StatusBadRequest)
			return
		case req.Branch == "" && req.ArtifactID == "":
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s' branch '%s': brand or artifact id not specified", req.Service, req.Environment, req.ArtifactID, req.Branch)
			Error(w, "branch or artifact id must be specified.", http.StatusBadRequest)
			return
		case req.Branch != "":
			logger.Infof("http: release: service '%s' environment '%s' branch '%s': releasing branch", req.Service, req.Environment, req.Branch)
			releaseID, err = flowSvc.ReleaseBranch(ctx, flow.Actor{
				Name:  req.CommitterName,
				Email: req.CommitterEmail,
			}, req.Environment, req.Service, req.Branch)
		case req.ArtifactID != "":
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': releasing artifact", req.Service, req.Environment, req.ArtifactID)
			releaseID, err = flowSvc.ReleaseArtifactID(ctx, flow.Actor{
				Name:  req.CommitterName,
				Email: req.CommitterEmail,
			}, req.Environment, req.Service, req.ArtifactID)
		default:
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s' branch '%s': neither brand nor artifact id specified", req.Service, req.Environment, req.ArtifactID, req.Branch)
			Error(w, "either branch or artifact id must be specified", http.StatusBadRequest)
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
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release skipped: environment up to date", req.Service, req.Environment, req.Branch, req.ArtifactID)
			case git.ErrArtifactNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: artifact not found", req.Service, req.Environment, req.Branch, req.ArtifactID)
				Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				return
			case git.ErrBranchBehindOrigin:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				Error(w, fmt.Sprintf("could not release right now. Please try again in a moment."), http.StatusServiceUnavailable)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: artifact not found", req.Service, req.Environment, req.Branch, req.ArtifactID)
				if req.Branch != "" {
					Error(w, fmt.Sprintf("artifact for branch '%s' not found for service '%s'", req.Branch, req.Service), http.StatusBadRequest)
				} else {
					Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				}
				return
			case flow.ErrUnknownConfiguration:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: source configuration not found", req.Service, req.Environment, req.Branch, req.ArtifactID)
				Error(w, fmt.Sprintf("configuration for environment '%s' not found for service '%s'. Is the environment specified in 'shuttle.yaml'?", req.Environment, req.Service), http.StatusBadRequest)
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

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

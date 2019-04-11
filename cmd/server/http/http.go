package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gopkg.in/go-playground/webhooks.v5/github"
)

type Options struct {
	Port                int
	Timeout             time.Duration
	GithubWebhookSecret string
	HamCtlAuthToken     string
	DaemonAuthToken     string
	SlackAuthToken      string

	ConfigRepo        string
	ArtifactFileName  string
	SSHPrivateKeyPath string
}

func NewServer(opts *Options, client *slack.Client) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", authenticate(opts.HamCtlAuthToken, promote(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.SlackAuthToken)))
	mux.HandleFunc("/release", authenticate(opts.HamCtlAuthToken, release(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.SlackAuthToken)))
	mux.HandleFunc("/status", authenticate(opts.HamCtlAuthToken, status(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath)))
	mux.HandleFunc("/policies", authenticate(opts.HamCtlAuthToken, policy(opts.ConfigRepo, opts.SSHPrivateKeyPath)))
	mux.HandleFunc("/webhook/github", githubWebhook(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.GithubWebhookSecret, opts.SlackAuthToken))
	mux.HandleFunc("/webhook/daemon", authenticate(opts.DaemonAuthToken, daemonWebhook(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, client)))

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

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func status(configRepo, artifactFileName, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		service := values.Get("servce")
		if emptyString(service) {
			requiredQueryError(w, "service")
			return
		}

		logger := log.WithFields("configRepo", configRepo, "artifactFileName", artifactFileName, "service", service)
		s, err := flow.Status(r.Context(), configRepo, artifactFileName, service, sshPrivateKeyPath)
		if err != nil {
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

		err = json.NewEncoder(w).Encode(httpinternal.StatusResponse{
			Dev:     &dev,
			Staging: &staging,
			Prod:    &prod,
		})
		if err != nil {
			logger.Errorf("http: status: service '%s': marshal response failed: %v", service, err)
		}
	}
}

func daemonWebhook(configRepo, artifactFileName, sshPrivateKeyPath string, slackClient *slack.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var podNotify httpinternal.PodNotifyRequest

		err := decoder.Decode(&podNotify)
		if err != nil {
			log.Errorf("http: daemon webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		err = flow.NotifyCommitter(context.Background(), configRepo, artifactFileName, sshPrivateKeyPath, &podNotify, slackClient)
		if err != nil {
			log.Errorf("http: daemon webhook failed: notify committer: %v", err)
			unknownError(w)
			return
		}

		log.WithFields("pod", podNotify.Name,
			"namespace", podNotify.Namespace,
			"state", podNotify.State,
			"message", podNotify.Message,
			"reason", podNotify.Reason,
			"artifactId", podNotify.ArtifactID,
			"logs", podNotify.Logs).Infof("Pod event received: %s, state=%s", podNotify.Name, podNotify.State)
	}
}

func githubWebhook(configRepo, artifactFileName, sshPrivateKeyPath, githubWebhookSecret, slackToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hook, _ := github.New(github.Options.Secret(githubWebhookSecret))
		payload, err := hook.Parse(r, github.PushEvent)
		if err != nil {
			log.Debugf("webhook: parse webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch payload.(type) {

		case github.PushPayload:
			push := payload.(github.PushPayload)
			if !isBranchPush(push.Ref) {
				w.WriteHeader(http.StatusOK)
				return
			}
			rgx := regexp.MustCompile(`\[(.*?)\]`)
			matches := rgx.FindStringSubmatch(push.HeadCommit.Message)
			if len(matches) < 2 {
				log.Debugf("webhook: no service match from commit '%s'", push.HeadCommit.Message)
				w.WriteHeader(http.StatusOK)
				return
			}
			serviceName := matches[1]

			// locate branch of commit
			branch, ok := branchName(push.HeadCommit.Modified, artifactFileName, serviceName)
			if !ok {
				log.Debugf("webhook: branch name not found: service '%s'", serviceName)
				w.WriteHeader(http.StatusOK)
				return
			}

			// lookup policies for branch
			autoReleases, err := policyinternal.GetAutoReleases(context.Background(), configRepo, sshPrivateKeyPath, serviceName, branch)
			if err != nil {
				log.Errorf("webhook: get auto release policies: service '%s' branch '%s': %v", serviceName, branch, err)
				Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			log.Debugf("webhook: found %d release policies: service '%s' branch '%s'", len(autoReleases), serviceName, branch)
			var errs error
			for _, autoRelease := range autoReleases {
				releaseID, err := flow.ReleaseBranch(context.Background(), configRepo, artifactFileName, serviceName, autoRelease.Environment, autoRelease.Branch, push.HeadCommit.Author.Name, push.HeadCommit.Author.Email, sshPrivateKeyPath, slackToken)
				if err != nil {
					if errors.Cause(err) != git.ErrNothingToCommit {
						errs = multierr.Append(errs, err)
						continue
					}
					log.WithFields("service", serviceName,
						"branch", branch,
						"environment", autoRelease.Environment).
						Infof("webhook: auto-release service '%s' from policy '%s' to '%s': nothing to commit", serviceName, autoRelease.ID, autoRelease.Environment)
					continue
				}
				log.WithFields("service", serviceName,
					"branch", branch,
					"environment", autoRelease.Environment,
					"commit", push.HeadCommit).
					Infof("webhook: auto-release service '%s' from policy '%s' of %s to %s", serviceName, autoRelease.ID, releaseID, autoRelease.Environment)
			}
			if errs != nil {
				log.Errorf("webhook: auto-release failed with one or more errors: service '%s' branch '%s' commit '%s': %v", serviceName, branch, push.HeadCommit.ID, errs)
				Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			log.Infof("webhook: handled successfully: service '%s' branch '%s' commit '%s'", serviceName, branch, push.HeadCommit.ID)
			w.WriteHeader(http.StatusOK)
			return
		default:
			log.Infof("webhook: payload type: default case hit: %v", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}

func isBranchPush(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/")
}

// branchName returns the branch name and a bool indicating one is found from a
// list of modified file paths.
//
// It only handles files that originates from a build operation, ie. non-build
// commits cannot be extracted.
func branchName(modifiedFiles []string, artifactFileName, svc string) (string, bool) {
	var branch string
	for _, f := range modifiedFiles {
		if !strings.Contains(f, artifactFileName) {
			continue
		}
		branch = strings.TrimPrefix(f, fmt.Sprintf("artifacts/%s/", svc))
		break
	}
	if len(branch) == 0 {
		return "", false
	}
	return strings.TrimSuffix(branch, fmt.Sprintf("/%s", artifactFileName)), true
}

func promote(configRepo, artifactFileName, sshPrivateKeyPath, slackToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.PromoteRequest

		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("Decode request body failed: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		releaseID, err := flow.Promote(r.Context(), configRepo, artifactFileName, req.Service, req.Environment, req.CommitterName, req.CommitterEmail, sshPrivateKeyPath, slackToken)

		var statusString string
		if err != nil {
			switch errors.Cause(err) {
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
			case flow.ErrUnknownEnvironment:
				log.Errorf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v", configRepo, artifactFileName, req.Service, req.Environment, err)
				Error(w, fmt.Sprintf("Unknown environment: %s", req.Environment), http.StatusBadRequest)
				return
			case artifact.ErrFileNotFound:
				Error(w, fmt.Sprintf("artifact not found for service '%s'", req.Service), http.StatusBadRequest)
				return
			default:
				log.Errorf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v", configRepo, artifactFileName, req.Service, req.Environment, err)
				Error(w, "Unknown error", http.StatusInternalServerError)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(httpinternal.PromoteResponse{
			Service:         req.Service,
			FromEnvironment: fromEnvironment,
			ToEnvironment:   req.Environment,
			Tag:             releaseID,
			Status:          statusString,
		})
		if err != nil {
			http.Error(w, "json encoding failed", http.StatusInternalServerError)
			return
		}
	}
}

func release(configRepo, artifactFileName, sshPrivateKeyPath, slackToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.ReleaseRequest

		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("Decode request body failed: %v", err)
			Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}
		log.Infof("http release: service '%s' environment '%s' branch '%s' artifact id '%s'", req.Service, req.Environment, req.Branch, req.ArtifactID)
		ctx := r.Context()
		var releaseID string
		switch {
		case req.Branch != "" && req.ArtifactID != "":
			Error(w, "Branch and artifact id cannot both be specified. Pick one", http.StatusBadRequest)
			return
		case req.Branch == "" && req.ArtifactID == "":
			Error(w, "Branch or artifact id must be specified.", http.StatusBadRequest)
			return
		case req.Branch != "":
			log.Infof("Release '%s' from branch '%s' to '%s'", req.Service, req.Branch, req.Environment)
			releaseID, err = flow.ReleaseBranch(ctx, configRepo, artifactFileName, req.Service, req.Environment, req.Branch, req.CommitterName, req.CommitterEmail, sshPrivateKeyPath, slackToken)
		case req.ArtifactID != "":
			releaseID, err = flow.ReleaseArtifactID(ctx, configRepo, artifactFileName, req.Service, req.Environment, req.ArtifactID, req.CommitterName, req.CommitterEmail, sshPrivateKeyPath, slackToken)
		default:
			Error(w, "Either branch or artifact id must be specified", http.StatusBadRequest)
			return
		}
		var statusString string
		if err != nil {
			cause := errors.Cause(err)
			switch cause {
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
				log.Info("release: nothing to commit")
			case git.ErrArtifactNotFound:
				Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				return
			case artifact.ErrFileNotFound:
				if req.Branch != "" {
					Error(w, fmt.Sprintf("artifact for branch '%s' not found for service '%s'", req.Branch, req.Service), http.StatusBadRequest)
				} else {
					Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				}
				return
			default:
				log.Errorf("http release flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v", configRepo, artifactFileName, req.Service, req.Environment, err)
				Error(w, "release flow failed", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(httpinternal.ReleaseResponse{
			Service:       req.Service,
			ReleaseID:     releaseID,
			ToEnvironment: req.Environment,
			Tag:           releaseID,
			Status:        statusString,
		})
		if err != nil {
			log.Errorf("release: marshal response failed: %v", err)
			Error(w, "unknown error", http.StatusInternalServerError)
		}
	}
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

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
	Port                 int
	Timeout              time.Duration
	GithubWebhookSecret  string
	HamCtlAuthToken      string
	DaemonAuthToken      string
	SlackAuthToken       string
	GrafanaDevAPIKey     string
	GrafanaStagingAPIKey string
	GrafanaProdAPIKey    string
	GrafanaDevUrl        string
	GrafanaStagingUrl    string
	GrafanaProdUrl       string

	UserMappings map[string]string

	ConfigRepo        string
	ArtifactFileName  string
	SSHPrivateKeyPath string
}

func NewServer(opts *Options, client *slack.Client) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", authenticate(opts.HamCtlAuthToken, promote(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.SlackAuthToken, opts.GrafanaDevAPIKey, opts.GrafanaStagingAPIKey, opts.GrafanaProdAPIKey, opts.GrafanaDevUrl, opts.GrafanaStagingUrl, opts.GrafanaProdUrl)))
	mux.HandleFunc("/release", authenticate(opts.HamCtlAuthToken, release(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.SlackAuthToken, opts.GrafanaDevAPIKey, opts.GrafanaStagingAPIKey, opts.GrafanaProdAPIKey, opts.GrafanaDevUrl, opts.GrafanaStagingUrl, opts.GrafanaProdUrl)))
	mux.HandleFunc("/status", authenticate(opts.HamCtlAuthToken, status(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath)))
	mux.HandleFunc("/rollback", authenticate(opts.HamCtlAuthToken, rollback(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.SlackAuthToken, opts.GrafanaDevAPIKey, opts.GrafanaStagingAPIKey, opts.GrafanaProdAPIKey, opts.GrafanaDevUrl, opts.GrafanaStagingUrl, opts.GrafanaProdUrl)))
	mux.HandleFunc("/policies", authenticate(opts.HamCtlAuthToken, policy(opts.ConfigRepo, opts.SSHPrivateKeyPath)))
	mux.HandleFunc("/webhook/github", githubWebhook(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, opts.GithubWebhookSecret, opts.SlackAuthToken, opts.GrafanaDevAPIKey, opts.GrafanaStagingAPIKey, opts.GrafanaProdAPIKey, opts.GrafanaDevUrl, opts.GrafanaStagingUrl, opts.GrafanaProdUrl))
	mux.HandleFunc("/webhook/daemon", authenticate(opts.DaemonAuthToken, daemonWebhook(opts.ConfigRepo, opts.ArtifactFileName, opts.SSHPrivateKeyPath, client, opts.UserMappings)))

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
		service := values.Get("service")
		if emptyString(service) {
			requiredQueryError(w, "service")
			return
		}

		logger := log.WithFields("configRepo", configRepo, "artifactFileName", artifactFileName, "service", service)
		ctx := r.Context()
		s, err := flow.Status(ctx, configRepo, artifactFileName, service, sshPrivateKeyPath)
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

func rollback(configRepo, artifactFileName, sshPrivateKeyPath, slackToken, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			Error(w, "not found", http.StatusNotFound)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.RollbackRequest
		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("http: rollback failed: decode request body: %v", err)
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

		logger := log.WithFields("configRepo", configRepo, "artifactFileName", artifactFileName, "service", req.Service, "req", req)
		ctx := r.Context()
		res, err := flow.Rollback(ctx, flow.FlowOptions{
			ConfigRepoURL:     configRepo,
			ArtifactFileName:  artifactFileName,
			Service:           req.Service,
			Environment:       req.Environment,
			CommitterName:     req.CommitterName,
			CommitterEmail:    req.CommitterEmail,
			SSHPrivateKeyPath: sshPrivateKeyPath,
			SlackToken:        slackToken,
			GrafanaAPIKey:     getGrafanaVarForEnv(req.Environment, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd),
			GrafanaUrl:        getGrafanaVarForEnv(req.Environment, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl),
		})
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: rollback cancelled: env '%s' service '%s'", req.Environment, req.Service)
				cancelled(w)
				return
			}
			switch errors.Cause(err) {
			case git.ErrReleaseNotFound:
				logger.Infof("http: rollback rejected: env '%s' service '%s': %v", req.Environment, req.Service, err)
				Error(w, fmt.Sprintf("no release of service '%s' available for rollback in environment '%s'", req.Service, req.Environment), http.StatusBadRequest)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(httpinternal.RollbackResponse{
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

func daemonWebhook(configRepo, artifactFileName, sshPrivateKeyPath string, slackClient *slack.Client, userMappings map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var podNotify httpinternal.PodNotifyRequest

		err := decoder.Decode(&podNotify)
		if err != nil {
			log.Errorf("http: daemon webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		err = flow.NotifyCommitter(context.Background(), configRepo, artifactFileName, sshPrivateKeyPath, &podNotify, slackClient, userMappings)
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

func githubWebhook(configRepo, artifactFileName, sshPrivateKeyPath, githubWebhookSecret, slackToken, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hook, _ := github.New(github.Options.Secret(githubWebhookSecret))
		payload, err := hook.Parse(r, github.PushEvent)
		if err != nil {
			log.Errorf("http: github webhook: decode request body failed: %v", err)
			invalidBodyError(w)
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
				log.Debugf("http: github webhook: no service match from commit '%s'", push.HeadCommit.Message)
				w.WriteHeader(http.StatusOK)
				return
			}
			serviceName := matches[1]

			// locate branch of commit
			branch, ok := branchName(push.HeadCommit.Modified, artifactFileName, serviceName)
			if !ok {
				log.Debugf("http: github webhook: service '%s': branch name not found", serviceName)
				w.WriteHeader(http.StatusOK)
				return
			}

			logger := log.WithFields("branch", branch, "service", serviceName, "commit", push.HeadCommit)
			// lookup policies for branch
			autoReleases, err := policyinternal.GetAutoReleases(context.Background(), configRepo, sshPrivateKeyPath, serviceName, branch)
			if err != nil {
				logger.Errorf("http: github webhook: service '%s' branch '%s': get auto release policies failed: %v", serviceName, branch, err)
				unknownError(w)
				return
			}
			logger.Debugf("http: github webhook: service '%s' branch '%s': found %d release policies", serviceName, branch, len(autoReleases))
			var errs error
			for _, autoRelease := range autoReleases {
				releaseID, err := flow.ReleaseBranch(context.Background(), flow.FlowOptions{
					ConfigRepoURL:     configRepo,
					ArtifactFileName:  artifactFileName,
					Service:           serviceName,
					Environment:       autoRelease.Environment,
					Branch:            autoRelease.Branch,
					CommitterName:     push.HeadCommit.Author.Name,
					CommitterEmail:    push.HeadCommit.Author.Email,
					SSHPrivateKeyPath: sshPrivateKeyPath,
					SlackToken:        slackToken,
					GrafanaAPIKey:     getGrafanaVarForEnv(autoRelease.Environment, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd),
					GrafanaUrl:        getGrafanaVarForEnv(autoRelease.Environment, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl),
				})
				if err != nil {
					if errors.Cause(err) != git.ErrNothingToCommit {
						errs = multierr.Append(errs, err)
						continue
					}
					logger.Infof("http: github webhook: service '%s': auto-release from policy '%s' to '%s': nothing to commit", serviceName, autoRelease.ID, autoRelease.Environment)
					continue
				}
				logger.Infof("http: github webhook: service '%s': auto-release from policy '%s' of %s to %s", serviceName, autoRelease.ID, releaseID, autoRelease.Environment)
			}
			if errs != nil {
				log.Errorf("http: github webhook: service '%s' branch '%s': auto-release failed with one or more errors: %v", serviceName, branch, errs)
				unknownError(w)
				return
			}
			log.Infof("http: github webhook: handled successfully: service '%s' branch '%s' commit '%s'", serviceName, branch, push.HeadCommit.ID)
			w.WriteHeader(http.StatusOK)
			return
		default:
			log.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
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

func promote(configRepo, artifactFileName, sshPrivateKeyPath, slackToken, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.PromoteRequest

		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("http: promote: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		logger := log.WithFields("configRepo", configRepo, "artifactFileName", artifactFileName, "service", req.Service, "req", req)
		ctx := r.Context()
		releaseID, err := flow.Promote(ctx, flow.FlowOptions{
			ConfigRepoURL:     configRepo,
			ArtifactFileName:  artifactFileName,
			Service:           req.Service,
			Environment:       req.Environment,
			CommitterName:     req.CommitterName,
			CommitterEmail:    req.CommitterEmail,
			SSHPrivateKeyPath: sshPrivateKeyPath,
			SlackToken:        slackToken,
			GrafanaAPIKey:     getGrafanaVarForEnv(req.Environment, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd),
			GrafanaUrl:        getGrafanaVarForEnv(req.Environment, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl),
		})

		var statusString string
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: promote: service '%s' environment '%s': promote cancelled", req.Service, req.Environment)
				cancelled(w)
				return
			}
			switch errors.Cause(err) {
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
				logger.Infof("http: promote: service '%s' environment '%s': promote skipped: environment up to date", req.Service, req.Environment)
			case flow.ErrUnknownEnvironment:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: %v", req.Service, req.Environment, err)
				Error(w, fmt.Sprintf("unknown environment: %s", req.Environment), http.StatusBadRequest)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: promote: service '%s' environment '%s': promote rejected: artifact not found", req.Service, req.Environment)
				Error(w, fmt.Sprintf("artifact not found for service '%s'", req.Service), http.StatusBadRequest)
				return
			default:
				logger.Infof("http: promote: service '%s' environment '%s': promote failed: %v", req.Service, req.Environment, err)
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
			logger.Errorf("http: promote: service '%s' environment '%s': marshal response failed: %v", req.Service, req.Environment, err)
		}
	}
}

func release(configRepo, artifactFileName, sshPrivateKeyPath, slackToken, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.ReleaseRequest

		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("http: release: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		ctx := r.Context()
		logger := log.WithFields(
			"configRepo", configRepo,
			"artifactFileName", artifactFileName,
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
			releaseID, err = flow.ReleaseBranch(ctx, flow.FlowOptions{
				ConfigRepoURL:     configRepo,
				ArtifactFileName:  artifactFileName,
				Service:           req.Service,
				Environment:       req.Environment,
				Branch:            req.Branch,
				CommitterName:     req.CommitterName,
				CommitterEmail:    req.CommitterEmail,
				SSHPrivateKeyPath: sshPrivateKeyPath,
				SlackToken:        slackToken,
				GrafanaAPIKey:     getGrafanaVarForEnv(req.Environment, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd),
				GrafanaUrl:        getGrafanaVarForEnv(req.Environment, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl),
			})
		case req.ArtifactID != "":
			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': releasing artifact", req.Service, req.Environment, req.ArtifactID)
			releaseID, err = flow.ReleaseArtifactID(ctx, flow.FlowOptions{
				ConfigRepoURL:     configRepo,
				ArtifactFileName:  artifactFileName,
				Service:           req.Service,
				Environment:       req.Environment,
				ArtifactID:        req.ArtifactID,
				CommitterName:     req.CommitterName,
				CommitterEmail:    req.CommitterEmail,
				SSHPrivateKeyPath: sshPrivateKeyPath,
				SlackToken:        slackToken,
				GrafanaAPIKey:     getGrafanaVarForEnv(req.Environment, grafanaApiKeyDev, grafanaApiKeyStaging, grafanaApiKeyProd),
				GrafanaUrl:        getGrafanaVarForEnv(req.Environment, grafanaDevUrl, grafanaStagingUrl, grafanaProdUrl),
			})
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
			cause := errors.Cause(err)
			switch cause {
			case git.ErrNothingToCommit:
				statusString = "Environment is already up-to-date"
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release skipped: environment up to date", req.Service, req.Environment, req.Branch, req.ArtifactID)
			case git.ErrArtifactNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: artifact not found", req.Service, req.Environment, req.Branch, req.ArtifactID)
				Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				return
			case artifact.ErrFileNotFound:
				logger.Infof("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release rejected: artifact not found", req.Service, req.Environment, req.Branch, req.ArtifactID)
				if req.Branch != "" {
					Error(w, fmt.Sprintf("artifact for branch '%s' not found for service '%s'", req.Branch, req.Service), http.StatusBadRequest)
				} else {
					Error(w, fmt.Sprintf("artifact '%s' not found for service '%s'", req.ArtifactID, req.Service), http.StatusBadRequest)
				}
				return
			default:
				logger.Errorf("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': release failed: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
				unknownError(w)
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
			logger.Errorf("http: release: service '%s' environment '%s' branch '%s' artifact id '%s': marshal response failed: %v", req.Service, req.Environment, req.Branch, req.ArtifactID, err)
		}
	}
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func getGrafanaVarForEnv(env, dev, staging, prod string) string {
	switch env {
	case "dev":
		return dev
	case "staging":
		return staging
	case "prod":
		return prod
	}
	return ""
}

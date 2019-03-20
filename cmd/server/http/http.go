package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func NewServer(port int, timeout time.Duration, configRepo, artifactFileName, sshPrivateKeyPath, githubWebhookSecret string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", promote(configRepo, artifactFileName, sshPrivateKeyPath))
	mux.HandleFunc("/status", status(configRepo, artifactFileName, sshPrivateKeyPath))
	mux.HandleFunc("/webhook", webhook(configRepo, artifactFileName, sshPrivateKeyPath, githubWebhookSecret))

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}
	log.Infof("Initializing HTTP Server on port %d", port)
	err := s.ListenAndServe()
	if err != nil {
		return errors.WithMessage(err, "listen and server")
	}
	return nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func status(configRepo, artifactFileName, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid := validateToken(r.Header.Get("Authorization"))
		if !valid {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		services, ok := r.URL.Query()["service"]
		if !ok || len(services[0]) == 0 {
			log.Errorf("query param service is missing for /status endpoint")
			http.Error(w, "Invalid query param", http.StatusBadRequest)
			return
		}
		service := services[0]

		s, err := flow.Status(configRepo, artifactFileName, service, sshPrivateKeyPath)
		if err != nil {
			log.Errorf("getting status failed: config repo '%s' artifact file name '%s' service '%s': %v", configRepo, artifactFileName, service, err)
			http.Error(w, "promote flow failed", http.StatusInternalServerError)
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
			log.Errorf("get status for service '%s' failed: marshal response: %v", service, err)
			http.Error(w, "unknown", http.StatusInternalServerError)
			return
		}
	}
}

func webhook(configRepo, artifactFileName, sshPrivateKeyPath, githubWebhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hook, _ := github.New(github.Options.Secret(githubWebhookSecret))
		payload, err := hook.Parse(r, github.PushEvent)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch payload.(type) {

		case github.PushPayload:
			push := payload.(github.PushPayload)
			rgx := regexp.MustCompile(`\[(.*?)\]`)
			matches := rgx.FindStringSubmatch(push.HeadCommit.Message)
			if len(matches) < 2 {
				log.Debugf("webhook: no service match from commit '%s'", push.HeadCommit.Message)
				w.WriteHeader(http.StatusOK)
				return
			}
			serviceName := matches[1]

			branch := "master"
			toEnvironment := "dev"
			for _, f := range push.HeadCommit.Modified {
				if !strings.Contains(f, branch) || !strings.Contains(f, artifactFileName) {
					continue
				}
				releaseID, err := flow.Promote(configRepo, artifactFileName, serviceName, toEnvironment, push.HeadCommit.Author.Name, push.HeadCommit.Author.Email, sshPrivateKeyPath)
				if err != nil {
					log.Errorf("webhook: promote failed: %v", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				log.WithFields("service", serviceName,
					"environment", toEnvironment,
					"commit", push.HeadCommit).Infof("auto-release of %s to %s", releaseID, toEnvironment)
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)

		default:
			log.Infof("webhook: payload type: default case hit: %v", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}

func promote(configRepo, artifactFileName, sshPrivateKeyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid := validateToken(r.Header.Get("Authorization"))
		if !valid {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var req httpinternal.PromoteRequest

		err := decoder.Decode(&req)
		if err != nil {
			log.Errorf("Decode request body failed: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		releaseID, err := flow.Promote(configRepo, artifactFileName, req.Service, req.Environment, req.CommitterName, req.CommitterEmail, sshPrivateKeyPath)

		var statusString string
		if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
			statusString = "Environment is already up-to-date"
		} else if err != nil && errors.Cause(err) == flow.ErrUnknownEnvironment {
			log.Errorf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v", configRepo, artifactFileName, req.Service, req.Environment, err)
			Error(w, fmt.Sprintf("Unknown environment: %s", req.Environment), http.StatusBadRequest)
		} else if err != nil {
			log.Errorf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v", configRepo, artifactFileName, req.Service, req.Environment, err)
			Error(w, "Unknown error", http.StatusInternalServerError)
			return
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

func validateToken(reqToken string) bool {
	serverToken := os.Getenv("RELEASE_MANAGER_AUTH_TOKEN")
	token := strings.TrimPrefix(reqToken, "Bearer ")

	if token == serverToken {
		return true
	}
	return false
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func Error(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(httpinternal.ErrorResponse{
		Message: message,
	})
	if err != nil {
		log.Errorf("json encoding failed in error response: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

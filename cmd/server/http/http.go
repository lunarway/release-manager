package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/pkg/errors"
)

func NewServer(port int, timeout time.Duration, configRepo, artifactFileName string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", promote(configRepo, artifactFileName))
	mux.HandleFunc("/status", status(configRepo, artifactFileName))

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	fmt.Printf("Initializing HTTP Server on port %d\n", port)
	err := s.ListenAndServe()
	if err != nil {
		return errors.WithMessage(err, "listen and server")
	}
	return nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

type PromoteRequest struct {
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
}

type PromoteResponse struct {
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	Status      string `json:"status,omitempty"`
}

func status(configRepo, artifactFileName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services, ok := r.URL.Query()["service"]

		if !ok || len(services[0]) == 0 {
			fmt.Printf("query param service is missing for /status endpoint: %v\n", ok)
			http.Error(w, "Invalid query param", http.StatusBadRequest)
			return
		}
		service := services[0]
		fmt.Printf("Service: %s\n", string(service))

		resp, err := flow.Status(configRepo, artifactFileName, service)
		if err != nil {
			fmt.Printf("getting status failed: config repo '%s' artifact file name '%s' service '%s': %v\n", configRepo, artifactFileName, service, err)
			http.Error(w, "promote flow failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			fmt.Printf("get status for service '%s' failed: marshal response: %v\n", service, err)
			http.Error(w, "unknown", http.StatusInternalServerError)
			return
		}
	}
}

func promote(configRepo, artifactFileName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)

		var req PromoteRequest

		err := decoder.Decode(&req)
		if err != nil {
			fmt.Printf("Decode request body failed: %v\n", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		var resp PromoteResponse
		resp.Service = req.Service
		resp.Environment = req.Environment

		fmt.Printf("Repo: %s, File: %s\n", configRepo, artifactFileName)
		_, err = flow.Promote(configRepo, artifactFileName, req.Service, req.Environment)

		if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
			resp.Status = "nothing to commit"
		} else if err != nil {
			fmt.Printf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v\n", configRepo, artifactFileName, req.Service, req.Environment, err)
			http.Error(w, "promote flow failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			http.Error(w, "json encoding failed", http.StatusInternalServerError)
			return
		}
	}
}

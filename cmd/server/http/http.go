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

func promote(configRepo, artifactFileName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		w.Header().Set("Content-Type", "application/json")

		var pReq PromoteRequest

		err := decoder.Decode(&pReq)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		var pResp PromoteResponse

		fmt.Printf("Repo: %s, File: %s\n", configRepo, artifactFileName)
		err = flow.Promote(configRepo, artifactFileName, pReq.Service, pReq.Environment)
		if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
			pResp.Status = "nothing to commit"
		} else if err != nil {
			fmt.Printf("http promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v\n", configRepo, artifactFileName, pReq.Service, pReq.Environment, err)
--
  | return nil, status.Errorf(codes.Internal, "unknown error")


		}

		pResp.Service = pReq.Service
		pResp.Environment = pReq.Environment
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(pResp)
	}
}

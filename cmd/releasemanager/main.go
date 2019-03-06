package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	gengrpc "github.com/lunarway/release-manager/generated/grpc"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	gRPCPort         = 7900
	port             = 8080
	timeout          = 20 * time.Second
	configRepo       = "git@github.com:lunarway/k8s-cluster-config.git"
	artifactFileName = "artifact.json"
)

type gRPCHandlers struct {
}

func (gRPCHandlers) Promote(ctx context.Context, req *gengrpc.PromoteRequest) (*gengrpc.PromoteResponse, error) {
	var pResp gengrpc.PromoteResponse

	err := flow.Promote(configRepo, artifactFileName, req.Service, req.Environment)
	if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
		pResp.Status = "nothing to commit"
	} else if err != nil {
		pResp.Status = err.Error()
	}

	pResp.Service = req.Service
	pResp.Environment = req.Environment

	return &pResp, nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", ping)
	mux.HandleFunc("/promote", promote)

	// Create GRPC server
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", gRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	gengrpc.RegisterReleaseManagerServer(grpcServer, &gRPCHandlers{})
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to initiate grpc: %v", err)
	}

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}
	s.ListenAndServe()

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

func promote(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	w.Header().Set("Content-Type", "application/json")

	var pReq PromoteRequest

	err := decoder.Decode(&pReq)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	var pResp PromoteResponse

	err = flow.Promote(configRepo, artifactFileName, pReq.Service, pReq.Environment)
	if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
		pResp.Status = "nothing to commit"
	} else if err != nil {
		pResp.Status = err.Error()
	}

	pResp.Service = pReq.Service
	pResp.Environment = pReq.Environment
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pResp)
}

package grpc

import (
	"context"
	"fmt"
	"log"
	"net"

	gengrpc "github.com/lunarway/release-manager/generated/grpc"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type grpcHandlers struct {
	ConfigRepo       string
	ArtifactFileName string
}

func NewServer(port int, configRepo, artifactFileName string) error {
	grpcServer := grpcpkg.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	gengrpc.RegisterReleaseManagerServer(grpcServer, &gRPCHandlers{
		ArtifactFileName: artifactFileName,
		ConfigRepo:       configRepo,
	})
	err = grpcServer.Serve(lis)
	if err != nil {
		return errors.WithMessage(err, "serve")
	}
	return nil
}

func (h gRPCHandlers) Promote(ctx context.Context, req *gengrpc.PromoteRequest) (*gengrpc.PromoteResponse, error) {
	var pResp gengrpc.PromoteResponse

	err := flow.Promote(h.ConfigRepo, h.ArtifactFileName, req.Service, req.Environment)
	if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
		pResp.Status = "nothing to commit"
	} else if err != nil {
		pResp.Status = err.Error()
	}

	pResp.Service = req.Service
	pResp.Environment = req.Environment

	return &pResp, nil
}

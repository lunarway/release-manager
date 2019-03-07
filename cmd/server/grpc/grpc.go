package grpc

import (
	"context"
	"fmt"
	"net"

	gengrpc "github.com/lunarway/release-manager/generated/grpc"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcHandlers struct {
	ConfigRepo       string
	ArtifactFileName string
}

func NewServer(port int, configRepo, artifactFileName string) error {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return errors.WithMessage(err, "listen on tcp port")
	}
	gengrpc.RegisterReleaseManagerServer(grpcServer, &grpcHandlers{
		ArtifactFileName: artifactFileName,
		ConfigRepo:       configRepo,
	})
	fmt.Printf("Initializing gRPC Server on port %d\n", port)
	err = grpcServer.Serve(lis)
	if err != nil {
		return errors.WithMessage(err, "serve")
	}
	return nil
}

func (h grpcHandlers) Promote(ctx context.Context, req *gengrpc.PromoteRequest) (*gengrpc.PromoteResponse, error) {
	var resp gengrpc.PromoteResponse
	resp.Service = req.Service
	resp.Environment = req.Environment

	err := flow.Promote(h.ConfigRepo, h.ArtifactFileName, req.Service, req.Environment)
	if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
		resp.Status = "nothing to commit"
	} else if err != nil {
		fmt.Printf("gRPC promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v\n", h.ConfigRepo, h.ArtifactFileName, req.Service, req.Environment, err)
		return nil, status.Errorf(codes.Internal, "unknown error")
	}

	return &resp, nil
}

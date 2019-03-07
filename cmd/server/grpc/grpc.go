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

func (h grpcHandlers) Status(ctx context.Context, req *gengrpc.StatusRequest) (*gengrpc.StatusResponse, error) {

	s, err := flow.Status(h.ConfigRepo, h.ArtifactFileName, req.Service)
	if err != nil {
		fmt.Printf("getting status failed: config repo '%s' artifact file name '%s' service '%s': %v\n", h.ConfigRepo, h.ArtifactFileName, req.Service, err)
		return nil, status.Errorf(codes.Internal, "unknown error")
	}

	fmt.Printf("Status: %v\n", s)

	// TODO: This is ugly, find a better way
	var resp gengrpc.StatusResponse
	dev := gengrpc.Environment{
		Message:   s.Dev.Message,
		Author:    s.Dev.Author,
		Tag:       s.Dev.Tag,
		Committer: s.Dev.Committer,
		Date:      s.Dev.Date.Unix(),
	}

	staging := gengrpc.Environment{
		Message:   s.Staging.Message,
		Author:    s.Staging.Author,
		Tag:       s.Staging.Tag,
		Committer: s.Staging.Committer,
		Date:      s.Staging.Date.Unix(),
	}

	prod := gengrpc.Environment{
		Message:   s.Prod.Message,
		Author:    s.Prod.Author,
		Tag:       s.Prod.Tag,
		Committer: s.Prod.Committer,
		Date:      s.Prod.Date.Unix(),
	}
	resp.Dev = &dev
	resp.Staging = &staging
	resp.Prod = &prod

	return &resp, nil
}

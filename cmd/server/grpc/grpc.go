package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

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

	releaseID, err := flow.Promote(h.ConfigRepo, h.ArtifactFileName, req.Service, req.Environment)

	var statusString string
	if err != nil && errors.Cause(err) == git.ErrNothingToCommit {
		statusString = "nothing to commit"
	} else if err != nil {
		fmt.Printf("gRPC promote flow failed: config repo '%s' artifact file name '%s' service '%s' environment '%s': %v\n", h.ConfigRepo, h.ArtifactFileName, req.Service, req.Environment, err)
		return nil, status.Errorf(codes.Internal, "unknown error")
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

	return &gengrpc.PromoteResponse{
		Service:         req.Service,
		FromEnvironment: fromEnvironment,
		ToEnvironment:   req.Environment,
		Tag:             releaseID,
		Status:          statusString,
	}, nil
}

func (h grpcHandlers) Status(ctx context.Context, req *gengrpc.StatusRequest) (*gengrpc.StatusResponse, error) {

	s, err := flow.Status(h.ConfigRepo, h.ArtifactFileName, req.Service)
	if err != nil {
		fmt.Printf("getting status failed: config repo '%s' artifact file name '%s' service '%s': %v\n", h.ConfigRepo, h.ArtifactFileName, req.Service, err)
		return nil, status.Errorf(codes.Internal, "unknown error")
	}

	dev := gengrpc.Environment{
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

	staging := gengrpc.Environment{
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

	prod := gengrpc.Environment{
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

	return &gengrpc.StatusResponse{
		Dev:     &dev,
		Staging: &staging,
		Prod:    &prod,
	}, nil
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

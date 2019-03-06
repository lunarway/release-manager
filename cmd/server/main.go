package main

import (
	"flag"
	"time"

	"github.com/lunarway/release-manager/cmd/server/grpc"
	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/pkg/errors"
)

func main() {
	var (
		gRPCPort         int
		HTTPPort         int
		timeout          time.Duration
		configRepo       string
		artifactFileName string
	)
	flag.StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "ssh url for the git config repository")
	flag.IntVar(&gRPCPort, "grpc-port", 7900, "the port of the grpc server")
	flag.IntVar(&HTTPPort, "http-port", 8080, "the port of the http server")
	flag.StringVar(&artifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	flag.DurationVar(&timeout, "timeout", (20 * time.Second), "timeout of both the grpc and http server")
	flag.Parse()

	done := make(chan error, 1)

	go func() {
		err := http.NewServer(HTTPPort, timeout, configRepo, artifactFileName)
		if err != nil {
			done <- errors.WithMessage(err, "new http server")
			return
		}
	}()

	go func() {
		err := grpc.NewServer(gRPCPort, configRepo, artifactFileName)
		if err != nil {
			done <- errors.WithMessage(err, "new grpc server")
			return
		}
	}()

	// Keep everything running
	select {}
}

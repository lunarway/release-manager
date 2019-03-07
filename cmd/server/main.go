package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunarway/release-manager/cmd/server/grpc"
	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/pkg/errors"
)

func main() {
	var (
		gRPCPort         int
		httpPort         int
		timeout          time.Duration
		configRepo       string
		artifactFileName string
	)
	flag.StringVar(&configRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "ssh url for the git config repository")
	flag.IntVar(&gRPCPort, "grpc-port", 7900, "the port of the grpc server")
	flag.IntVar(&HTTPPort, "http-port", 8080, "the port of the http server")
	flag.StringVar(&artifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	flag.DurationVar(&timeout, "timeout", 20 * time.Second, "timeout of both the grpc and http server")
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigs:
			done <- fmt.Errorf("received os signal '%s'", sig)
		}
	}()

	err := <-done
	if err != nil {
		fmt.Printf("Exited unknown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Program ended")
}

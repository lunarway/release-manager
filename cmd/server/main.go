package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/server/command"
	"github.com/lunarway/release-manager/internal/log"
)

var (
	version = ""
)

func main() {
	var logConfiguration *log.Configuration
	logConfiguration.ParseFromEnvironmnet()
	logger := log.New(logConfiguration)
	c, err := command.NewRoot(logger, version)
	if err != nil {
		logger.Errorf("Error: %v", err)
		os.Exit(1)
	}
	logConfiguration = log.RegisterFlags(c)
	err = c.Execute()
	if err != nil {
		logger.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

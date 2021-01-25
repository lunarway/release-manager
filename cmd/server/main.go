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
	log.Init(logConfiguration)
	c, err := command.NewRoot(version)
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}
	err = c.Execute()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

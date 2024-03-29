package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/artifact/command"
	"github.com/lunarway/release-manager/internal/log"
	"go.uber.org/zap/zapcore"
)

var (
	version = ""
)

func main() {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: false,
	})
	err := command.NewRoot(version).Execute()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

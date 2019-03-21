package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/daemon/command"
	"github.com/lunarway/release-manager/internal/log"
)

func main() {
	log.Init()
	err := command.DaemonCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}

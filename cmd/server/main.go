package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/server/command"
	"github.com/lunarway/release-manager/internal/log"
)

func main() {
	log.Init()
	err := command.NewCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}

package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/artifact/command"
)

func main() {
	err := command.NewCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}

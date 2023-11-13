package main

import (
	"os"

	"github.com/lunarway/release-manager/cmd/daemon/command"
)

var (
	version = ""
)

func main() {
	c, err := command.NewRoot(version)
	if err != nil {
		os.Exit(1)
	}
	err = c.Execute()
	if err != nil {
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
)

var (
	version = ""
)

func main() {
	c, err := command.NewRoot(&version)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	err = c.Execute()
	if err != nil {
		os.Exit(1)
	}
}

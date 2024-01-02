package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
	"github.com/lunarway/release-manager/internal/http"
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
	c.SilenceUsage = true
	c.SilenceErrors = true

	err = c.Execute()
	if err != nil {
		if errors.Is(err, http.ErrLoginRequired) {
			fmt.Printf("You are not logged in. To log in, please run the following command:\n 'hamctl login'\n")
		} else {
			fmt.Printf("Error: %v\n", err)
		}

		os.Exit(1)
	}
}

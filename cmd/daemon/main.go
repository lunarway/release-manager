package main

import (
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/daemon/command"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
)

var (
	version = ""
)

func main() {
	c, err := command.DaemonCommand()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "prints the version number of artifact",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	c.AddCommand(versionCmd)
	err = c.Execute()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"github.com/lunarway/release-manager/cmd/server/command"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
	"os"
)

var (
	version = ""
)

func main() {
	c, err := command.NewCommand()
	if err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(1)
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "prints the version number of hamctl",
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

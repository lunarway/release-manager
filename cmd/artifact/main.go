package main

import (
	"os"

	"fmt"

	"github.com/lunarway/release-manager/cmd/artifact/command"
	"github.com/spf13/cobra"
)

var (
	version = ""
)

func main() {
	c, err := command.NewCommand()
	if err != nil {
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
	c.Execute()
}

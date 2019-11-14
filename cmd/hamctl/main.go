package main

import (
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
	"github.com/spf13/cobra"
)

var (
	version = ""
)

func main() {
	c, err := command.NewCommand(&version)
	if err != nil {
		fmt.Printf("Error: %v", err)
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
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

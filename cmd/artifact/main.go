package main

import (
	"fmt"
	"os"

	"github.com/lunarway/release-manager/cmd/artifact/command"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

var (
	version = ""
)

func main() {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: false,
	})
	c, err := command.NewCommand()
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

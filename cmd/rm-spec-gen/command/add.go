package command

import (
	"path"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/spf13/cobra"
)

// NewCommand returns a new instance of a rm-gen-spec command.
func addCommand(options *Options) *cobra.Command {
	var command = &cobra.Command{
		Use:   "add",
		Short: "",
		Long:  "",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(appendTestSubCommand(options))
	// command.AddCommand(buildCommand(&options))
	// command.AddCommand(pushCommand(&options))
	// command.AddCommand(snykCodeCommand(&options))
	// command.AddCommand(snykDockerCommand(&options))
	return command
}

func appendTestSubCommand(options *Options) *cobra.Command {
	var testData spec.TestData
	var stage spec.Stage
	command := &cobra.Command{
		Use:   "test",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				stage.Name = "Test"
				stage.ID = "test"
				stage.Data = testData
				s.Stages = append(s.Stages, stage)
				return s
			})
		},
	}
	command.Flags().IntVar(&testData.TestResults.Passed, "passed", 0, "")
	command.Flags().IntVar(&testData.TestResults.Failed, "failed", 0, "")
	command.Flags().IntVar(&testData.TestResults.Skipped, "skipped", 0, "")
	command.Flags().StringVar(&testData.URL, "url", "", "")
	command.MarkFlagRequired("passed")
	command.MarkFlagRequired("failed")
	command.MarkFlagRequired("skipped")
	return command
}

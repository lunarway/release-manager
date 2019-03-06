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
		Short: "add sub commands",
		Long:  "",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(appendTestSubCommand(options))
	command.AddCommand(appendBuildSubCommand(options))
	command.AddCommand(appendPushSubCommand(options))
	command.AddCommand(appendSnykCodeSubCommand(options))
	command.AddCommand(appendSnykDockerSubCommand(options))
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
				return setStage(s, stage)
			})
		},
	}
	command.Flags().IntVar(&testData.Results.Passed, "passed", 0, "")
	command.Flags().IntVar(&testData.Results.Failed, "failed", 0, "")
	command.Flags().IntVar(&testData.Results.Skipped, "skipped", 0, "")
	command.Flags().StringVar(&testData.URL, "url", "", "")
	command.MarkFlagRequired("passed")
	command.MarkFlagRequired("failed")
	command.MarkFlagRequired("skipped")
	return command
}

func appendBuildSubCommand(options *Options) *cobra.Command {
	var buildData spec.BuildData
	var stage spec.Stage
	command := &cobra.Command{
		Use:   "build",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				stage.Name = "Build"
				stage.ID = "build"
				stage.Data = buildData
				return setStage(s, stage)
			})
		},
	}

	command.Flags().StringVar(&buildData.Image, "image", "", "")
	command.Flags().StringVar(&buildData.Tag, "tag", "", "")
	command.Flags().StringVar(&buildData.DockerVersion, "docker-version", "", "")
	command.MarkFlagRequired("image")
	command.MarkFlagRequired("tag")
	command.MarkFlagRequired("docker-version")
	return command
}

func appendSnykDockerSubCommand(options *Options) *cobra.Command {
	var snykDockerData spec.SnykDockerData
	var stage spec.Stage
	command := &cobra.Command{
		Use:   "snyk-docker",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				stage.Name = "Security Scan - Docker"
				stage.ID = "snyk-docker"
				stage.Data = snykDockerData
				return setStage(s, stage)
			})
		},
	}

	command.Flags().StringVar(&snykDockerData.BaseImage, "base-image", "", "")
	command.Flags().StringVar(&snykDockerData.SnykVersion, "snyk-version", "", "")
	command.Flags().StringVar(&snykDockerData.Tag, "tag", "", "")
	command.Flags().StringVar(&snykDockerData.URL, "url", "", "")
	command.Flags().IntVar(&snykDockerData.Vulnerabilities.High, "high", 0, "")
	command.Flags().IntVar(&snykDockerData.Vulnerabilities.Medium, "medium", 0, "")
	command.Flags().IntVar(&snykDockerData.Vulnerabilities.Low, "low", 0, "")
	return command
}

func appendSnykCodeSubCommand(options *Options) *cobra.Command {
	var snykCodeData spec.SnykCodeData
	var stage spec.Stage
	command := &cobra.Command{
		Use:   "snyk-code",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				stage.Name = "Security Scan - Code"
				stage.ID = "snyk-code"
				stage.Data = snykCodeData
				return setStage(s, stage)
			})
		},
	}

	command.Flags().StringVar(&snykCodeData.Language, "language", "", "")
	command.Flags().StringVar(&snykCodeData.SnykVersion, "snyk-version", "", "")
	command.Flags().StringVar(&snykCodeData.URL, "url", "", "")
	command.Flags().IntVar(&snykCodeData.Vulnerabilities.High, "high", 0, "")
	command.Flags().IntVar(&snykCodeData.Vulnerabilities.Medium, "medium", 0, "")
	command.Flags().IntVar(&snykCodeData.Vulnerabilities.Low, "low", 0, "")
	return command
}

func appendPushSubCommand(options *Options) *cobra.Command {
	var pushData spec.PushData
	var stage spec.Stage
	command := &cobra.Command{
		Use:   "push",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				stage.Name = "Push"
				stage.ID = "push"
				stage.Data = pushData
				return setStage(s, stage)
			})
		},
	}

	command.Flags().StringVar(&pushData.Image, "image", "", "")
	command.Flags().StringVar(&pushData.Tag, "tag", "", "")
	command.Flags().StringVar(&pushData.DockerVersion, "docker-version", "", "")
	command.MarkFlagRequired("image")
	command.MarkFlagRequired("tag")
	command.MarkFlagRequired("docker-version")
	return command
}

func setStage(s spec.Spec, stage spec.Stage) spec.Spec {
	var updatedStages []spec.Stage
	var replaced bool

	for i := range s.Stages {
		if s.Stages[i].ID == stage.ID {
			updatedStages = append(updatedStages, stage)
			replaced = true
			continue
		}
		updatedStages = append(updatedStages, s.Stages[i])
	}

	if !replaced {
		updatedStages = append(updatedStages, stage)
	}

	s.Stages = updatedStages
	return s
}

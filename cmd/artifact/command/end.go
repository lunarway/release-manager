package command

import (
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func endCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "end",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return artifact.Update(path.Join(options.RootPath, options.FileName), func(s artifact.Spec) artifact.Spec {
				s.CI.End = time.Now()
				return s
			})
		},
	}

	return command
}

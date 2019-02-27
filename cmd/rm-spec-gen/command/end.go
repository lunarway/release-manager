package command

import (
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/spf13/cobra"
)

// NewCommand sets up the move command
func endCommand(options *Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "end",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return spec.Update(path.Join(options.RootPath, options.FileName), func(s spec.Spec) spec.Spec {
				s.CI.End = time.Now()
				return s
			})
		},
	}

	return command
}

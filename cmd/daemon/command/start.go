package command

import (
	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	"github.com/spf13/cobra"
)

func StartDaemon() *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			kubectl, err := kubernetes.NewClient()
			if err != nil {
				return err
			}

			err = kubectl.WatchPods()
			if err != nil {
				return err
			}
			return nil
		},
	}

	return command
}

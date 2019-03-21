package command

import (
	"context"

	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	"github.com/lunarway/release-manager/internal/log"
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

			succeededFunc := func(event *kubernetes.PodEvent) error {
				//TODO: send event to release-manager
				log.WithFields("namespace", event.Namespace,
					"name", event.PodName,
					"exitCode", event.ExitCode,
					"reason", event.Reason,
				).Infof("Success: pod=%s", event.PodName)
				return nil
			}

			failedFunc := func(event *kubernetes.PodEvent) error {

				if event.Reason == "CrashLoopBackOff" {
					logs, err := kubectl.GetLogs(event.PodName, event.Namespace)
					if err != nil {
						return err
					}
					log.WithFields("logs", logs).Infof("CrashLoopBackOff Logs")
				}

				//TODO: send event to release-manager
				log.WithFields("namespace", event.Namespace,
					"name", event.PodName,
					"exitCode", event.ExitCode,
					"reason", event.Reason,
					"message", event.Message,
				).Infof("Failure: pod=%s, reason=%s", event.PodName, event.Reason)
				return nil
			}

			err = kubectl.WatchPods(context.Background(), succeededFunc, failedFunc)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return command
}

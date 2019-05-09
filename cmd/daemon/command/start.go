package command

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
)

func StartDaemon() *cobra.Command {
	var authToken, releaseManagerUrl, environment string
	var logConfiguration *log.Configuration
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			logConfiguration.ParseFromEnvironmnet()
			log.Init(logConfiguration)
			kubectl, err := kubernetes.NewClient()
			if err != nil {
				return err
			}

			succeededFunc := func(event *kubernetes.PodEvent) error {
				notifyReleaseManager(event, "", releaseManagerUrl, authToken, environment)
				return nil
			}

			failedFunc := func(event *kubernetes.PodEvent) error {
				if event.Reason == "CrashLoopBackOff" {
					logs, err := kubectl.GetLogs(event.Name, event.Namespace)
					if err != nil {
						return err
					}
					notifyReleaseManager(event, logs, releaseManagerUrl, authToken, environment)
					return nil
				}
				notifyReleaseManager(event, "", releaseManagerUrl, authToken, environment)
				return nil
			}

			for {
				err = kubectl.WatchPods(context.Background(), succeededFunc, failedFunc)
				if err != nil && err != kubernetes.ErrWatcherClosed {
					return err
				}
			}
		},
	}
	command.Flags().StringVar(&releaseManagerUrl, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&authToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	command.Flags().StringVar(&environment, "environment", "", "environment where release-daemon is running")
	command.MarkFlagRequired("environment")
	logConfiguration = log.RegisterFlags(command)
	return command
}

func notifyReleaseManager(event *kubernetes.PodEvent, logs, releaseManagerUrl, authToken, environment string) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(httpinternal.PodNotifyRequest{
		Name:        event.Name,
		Namespace:   event.Namespace,
		Message:     event.Message,
		Reason:      event.Reason,
		State:       event.State,
		Containers:  mapContainers(event.Containers),
		ArtifactID:  event.ArtifactID,
		Logs:        logs,
		Environment: environment,
	})

	if err != nil {
		log.Errorf("error encoding StatusNotifyRequest")
		return
	}

	url := releaseManagerUrl + "/webhook/daemon"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		log.Errorf("error generating PodNotifyRequest to %s", url)
		return
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error posting PodNotifyRequest to %s", url)
		return
	}
	if resp.StatusCode != 200 {
		log.Errorf("release-manager returned status-code in notify webhook: %d", resp.Status)
		return
	}
}

func mapContainers(containers []kubernetes.Container) []httpinternal.Container {
	h := make([]httpinternal.Container, len(containers))
	for i, c := range containers {
		h[i] = httpinternal.Container{
			Name:         c.Name,
			State:        c.State,
			Reason:       c.Reason,
			Message:      c.Message,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
		}
	}
	return h
}

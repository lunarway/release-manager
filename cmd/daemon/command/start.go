package command

import (
	"context"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/spf13/cobra"
	"io"
	"bytes"
	"net/http"
	"encoding/json"
	"time"
	"os"
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
				notifyReleaseManager(event, "")
				return nil
			}

			failedFunc := func(event *kubernetes.PodEvent) error {
				if event.Reason == "CrashLoopBackOff" {
					logs, err := kubectl.GetLogs(event.PodName, event.Namespace)
					if err != nil {
						return err
					}
					notifyReleaseManager(event, logs)
					return nil
				}
				notifyReleaseManager(event, "")
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

func notifyReleaseManager(event *kubernetes.PodEvent, logs string) {
	client := &http.Client{
		Timeout: 20*time.Second,
	}

	var b io.ReadWriter
	b = &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(httpinternal.StatusNotifyRequest{
		PodName: event.PodName,
		Namespace: event.Namespace,
		Message: event.Message,
		Reason: event.Reason,
		Status: event.Status,
		ArtifactID: event.ArtifactID,
		Logs: logs,
	})

	if err != nil {
		log.Errorf("error encoding StatusNotifyRequest")
	}

	url := os.Getenv("RELEASE_MANAGER_ADDRESS")+"/webhook/daemon"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		log.Errorf("error generating StatusNotifyRequest to %s", url)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("DAEMON_AUTH_TOKEN"))
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error posting StatusNotifyRequest to %s", url)
	}
	if resp.StatusCode != 200 {
		log.Errorf("release-manager returned status-code in notify webhook: %d", resp.StatusCode)
	}
}
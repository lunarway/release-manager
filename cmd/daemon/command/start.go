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
	var authToken, releaseManagerUrl string
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			kubectl, err := kubernetes.NewClient()
			if err != nil {
				return err
			}

			succeededFunc := func(event *kubernetes.PodEvent) error {
				notifyReleaseManager(event, "", releaseManagerUrl, authToken)
				return nil
			}

			failedFunc := func(event *kubernetes.PodEvent) error {
				if event.Reason == "CrashLoopBackOff" {
					logs, err := kubectl.GetLogs(event.Name, event.Namespace)
					if err != nil {
						return err
					}
					notifyReleaseManager(event, logs, releaseManagerUrl, authToken)
					return nil
				}
				notifyReleaseManager(event, "", releaseManagerUrl, authToken)
				return nil
			}

			err = kubectl.WatchPods(context.Background(), succeededFunc, failedFunc)
			if err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&releaseManagerUrl, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&authToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	return command
}

func notifyReleaseManager(event *kubernetes.PodEvent, logs, releaseManagerUrl, authToken string) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(httpinternal.StatusNotifyRequest{
		PodName:    event.Name,
		Namespace:  event.Namespace,
		Message:    event.Message,
		Reason:     event.Reason,
		Status:     event.State,
		ArtifactID: event.ArtifactID,
		Logs:       logs,
	})

	if err != nil {
		log.Errorf("error encoding StatusNotifyRequest")
		return
	}

	url := releaseManagerUrl + "/webhook/daemon"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		log.Errorf("error generating StatusNotifyRequest to %s", url)
		return
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error posting StatusNotifyRequest to %s", url)
		return
	}
	if resp.StatusCode != 200 {
		log.Errorf("release-manager returned status-code in notify webhook: %d", resp.Status)
		return
	}
}

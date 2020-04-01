package command

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunarway/release-manager/cmd/daemon/flux"
	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func StartDaemon() *cobra.Command {
	var authToken, releaseManagerUrl, environment, kubeConfigPath, fluxApiBinding string
	var logConfiguration *log.Configuration
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)

			logConfiguration.ParseFromEnvironmnet()
			log.Init(logConfiguration)

			kubectl, err := kubernetes.NewClient(kubeConfigPath)
			if err != nil {
				return err
			}

			log.Info("Deamon started")

			go func() {
				api := flux.NewAPI(&flux.ReleaseManagerExporter{
					Log:         log.With("type", "exporter"),
					Url:         releaseManagerUrl,
					AuthToken:   authToken,
					Environment: environment,
				}, log.With("type", "api"))

				flux.HandleWebsocket(api)
				flux.HandleV6(api)

				err = api.Listen(fluxApiBinding)
				if err != nil {
					done <- errors.WithMessage(err, "flux-api: listen err")
					return
				}
			}()

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

			go func() {
				for {
					err = kubectl.WatchPods(context.Background(), succeededFunc, failedFunc)
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl watcher: watcher closed")
						return
					}
				}
			}()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigs
				log.Infof("received os signal '%s'", sig)
				done <- nil
			}()

			err = <-done
			if err != nil {
				log.Errorf("Exited unknown error: %v", err)
				os.Exit(1)
			}
			log.Infof("Program ended")
			return nil
		},
	}
	command.Flags().StringVar(&releaseManagerUrl, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&authToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	command.Flags().StringVar(&environment, "environment", "", "environment where release-daemon is running")
	command.Flags().StringVar(&kubeConfigPath, "kubeconfig", "", "path to kubeconfig file. If not specified, then daemon is expected to run inside kubernetes")
	command.Flags().StringVar(&fluxApiBinding, "flux-api-binding", ":8080", "binding of the daemon flux api server")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
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
		Name:           event.Name,
		Namespace:      event.Namespace,
		Message:        event.Message,
		Reason:         event.Reason,
		State:          event.State,
		Containers:     mapContainers(event.Containers),
		ArtifactID:     event.ArtifactID,
		Logs:           logs,
		Environment:    environment,
		AuthorEmail:    event.AuthorEmail,
		CommitterEmail: event.CommitterEmail,
	})

	if err != nil {
		log.Errorf("error encoding StatusNotifyRequest: %+v", err)
		return
	}

	url := releaseManagerUrl + "/webhook/daemon"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		log.Errorf("error generating PodNotifyRequest to %s: %+v", url, err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error posting PodNotifyRequest to %s: %+v", url, err)
		return
	}
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read response body: %+v", err)
		}
		log.Errorf("release-manager returned %s status-code in notify webhook: %s", resp.Status, body)
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

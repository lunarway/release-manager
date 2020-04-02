package command

import (
	"context"
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

func StartDaemon(version string) *cobra.Command {
	var environment, kubeConfigPath, fluxApiBinding string
	var logConfiguration *log.Configuration

	client := httpinternal.Client{
		Metadata: httpinternal.Metadata{
			CLIVersion: version,
		},
	}
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
					Client:      client,
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
				err = notifyReleaseManager(&client, event, "", environment)
				if err != nil {
					return err
				}
				return nil
			}

			failedFunc := func(event *kubernetes.PodEvent) error {
				if event.Reason == "CrashLoopBackOff" {
					logs, err := kubectl.GetLogs(event.Name, event.Namespace)
					if err != nil {
						return err
					}
					err = notifyReleaseManager(&client, event, logs, environment)
					if err != nil {
						return err
					}
					return nil
				}
				err = notifyReleaseManager(&client, event, "", environment)
				if err != nil {
					return err
				}
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
	command.Flags().StringVar(&client.BaseURL, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&client.Metadata.AuthToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	command.Flags().DurationVar(&client.Timeout, "http-timeout", 20*time.Second, "HTTP request timeout")
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

func notifyReleaseManager(client *httpinternal.Client, event *kubernetes.PodEvent, logs, environment string) error {
	var resp httpinternal.PodNotifyResponse
	url, err := client.URL("webhook/daemon")
	if err != nil {
		return err
	}
	err = client.Do(http.MethodPost, url, httpinternal.PodNotifyRequest{
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
	}, &resp)
	if err != nil {
		return err
	}
	return nil
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

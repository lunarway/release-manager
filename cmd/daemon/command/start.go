package command

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Release Daemon wathces for the following changes
// 1. Successful release of a new deployment
// 2. Detects CrashLoopBackOff, fetches the specific pods log
// 3. Detects CreateContainerConfigError, and fetches the message about the wrong config.
func StartDaemon() *cobra.Command {
	var environment, kubeConfigPath string
	var moduloCrashReportNotif float64
	var logConfiguration *log.Configuration
	var clientConfig http.Config
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)

			logConfiguration.ParseFromEnvironmnet()
			log.Init(logConfiguration)

			client, auth := http.NewClient(&clientConfig)

			kubectl, err := kubernetes.NewClient(kubeConfigPath, moduloCrashReportNotif, &kubernetes.ReleaseManagerExporter{
				Log:         log.With("type", "k8s-exporter"),
				Client:      client.Webhook,
				ClientAuth:  auth,
				Environment: environment,
			})
			if err != nil {
				return err
			}

			log.Info("Deamon started")

			go func() {
				for {
					err = kubectl.HandleNewDeployments(context.Background())
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl handle new deployments: watcher closed")
						return
					}
				}
			}()

			go func() {
				for {
					err = kubectl.HandleNewDaemonSets(context.Background())
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl handle new daemonsets: watcher closed")
						return
					}
				}
			}()

			go func() {
				for {
					err = kubectl.HandleNewStatefulSets(context.Background())
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl handle new statefulsets: watcher closed")
						return
					}
				}
			}()

			go func() {
				for {
					err = kubectl.HandlePodErrors(context.Background())
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl handle pod errors: watcher closed")
						return
					}
				}
			}()

			go func() {
				for {
					err = kubectl.HandleJobErrors(context.Background())
					if err != nil && err != kubernetes.ErrWatcherClosed {
						done <- errors.WithMessage(err, "kubectl handle job errors: watcher closed")
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
	command.Flags().StringVar(&clientConfig.BaseURL, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&clientConfig.AuthToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	// command.Flags().DurationVar(&clientConfig.Timeout, "http-timeout", 20*time.Second, "HTTP request timeout")
	command.Flags().StringVar(&environment, "environment", "", "environment where release-daemon is running")
	command.Flags().StringVar(&kubeConfigPath, "kubeconfig", "", "path to kubeconfig file. If not specified, then daemon is expected to run inside kubernetes")
	command.Flags().Float64Var(&moduloCrashReportNotif, "modulo-crash-report-notif", 5, "modulo for how often to report CrashLoopBackOff events")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("environment")
	logConfiguration = log.RegisterFlags(command)
	return command
}

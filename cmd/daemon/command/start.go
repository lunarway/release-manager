package command

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	flux_notification_controller "github.com/lunarway/release-manager/cmd/daemon/flux2notifications"
	"github.com/lunarway/release-manager/cmd/daemon/kubernetes"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/cache"
)

// Release Daemon wathces for the following changes
// 1. Successful release of a new deployment
// 2. Detects CrashLoopBackOff, fetches the specific pods log
// 3. Detects CreateContainerConfigError, and fetches the message about the wrong config.
func StartDaemon(logger *log.Logger) *cobra.Command {
	var environment, kubeConfigPath string
	var moduloCrashReportNotif float64

	client := httpinternal.Client{}
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-daemon",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)

			exporter := &kubernetes.ReleaseManagerExporter{
				Log:         logger.With("type", "k8s-exporter"),
				Client:      client,
				Environment: environment,
			}

			kubectl, err := kubernetes.NewClient(logger, kubeConfigPath)
			if err != nil {
				return err
			}

			handlerFactory := func(handlers cache.ResourceEventHandlerFuncs) cache.ResourceEventHandler {
				return kubernetes.ResourceEventHandlerFuncs{
					ShouldProcess:             kubectl.HasSynced,
					ResourceEventHandlerFuncs: handlers,
				}
			}

			kubernetes.RegisterDeploymentInformer(logger, kubectl.InformerFactory, exporter, handlerFactory, kubectl.Clientset)
			kubernetes.RegisterDaemonSetInformer(logger, kubectl.InformerFactory, exporter, handlerFactory, kubectl.Clientset)
			kubernetes.RegisterJobInformer(logger, kubectl.InformerFactory, exporter, handlerFactory, kubectl.Clientset)
			kubernetes.RegisterPodInformer(logger, kubectl.InformerFactory, exporter, handlerFactory, kubectl.Clientset, moduloCrashReportNotif)
			kubernetes.RegisterStatefulSetInformer(logger, kubectl.InformerFactory, exporter, handlerFactory, kubectl.Clientset)
			server := flux_notification_controller.NewHttpServer(logger)
			go func() {
				err := server.ListenAndServe()
				if err != nil {
					done <- errors.WithMessage(err, "start notification server")
				}
			}()

			logger.Info("Deamon started")

			stopCh := make(chan struct{})
			err = kubectl.Start(stopCh)
			if err != nil {
				return errors.WithMessage(err, "could not start client")
			}

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigs
				logger.Infof("received os signal '%s'", sig)
				done <- nil
			}()

			err = <-done
			if err != nil {
				logger.Errorf("Exited unknown error: %v", err)
				os.Exit(1)
			}

			err = server.Close()
			if err != nil {
				logger.Errorf("Failed to close the notification server: %s", err)
				os.Exit(1)
			}

			logger.Infof("Program ended")
			return nil
		},
	}
	command.Flags().StringVar(&client.BaseURL, "release-manager-url", os.Getenv("RELEASE_MANAGER_ADDRESS"), "address of the release-manager, e.g. http://release-manager")
	command.Flags().StringVar(&client.Metadata.AuthToken, "auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "token to be used to communicate with the release-manager")
	command.Flags().DurationVar(&client.Timeout, "http-timeout", 20*time.Second, "HTTP request timeout")
	command.Flags().StringVar(&environment, "environment", "", "environment where release-daemon is running")
	command.Flags().StringVar(&kubeConfigPath, "kubeconfig", "", "path to kubeconfig file. If not specified, then daemon is expected to run inside kubernetes")
	command.Flags().Float64Var(&moduloCrashReportNotif, "modulo-crash-report-notif", 5, "modulo for how often to report CrashLoopBackOff events")
	// errors are skipped here as the only case they can occour are if thee flag
	// does not exist on the command.
	//nolint:errcheck
	command.MarkFlagRequired("environment")
	return command
}

package command

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunarway/release-manager/cmd/server/grpc"
	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStart(options *Options) *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-manager",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)

			go func() {
				err := http.NewServer(options.httpPort, options.timeout, options.configRepo, options.artifactFileName, options.sshPrivateKeyPath)
				if err != nil {
					done <- errors.WithMessage(err, "new http server")
					return
				}
			}()

			go func() {
				err := grpc.NewServer(options.grpcPort, options.configRepo, options.artifactFileName)
				if err != nil {
					done <- errors.WithMessage(err, "new grpc server")
					return
				}
			}()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				select {
				case sig := <-sigs:
					done <- fmt.Errorf("received os signal '%s'", sig)
				}
			}()

			err := <-done
			if err != nil {
				fmt.Printf("Exited unknown error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Program ended")
			return nil
		},
	}

	return command
}

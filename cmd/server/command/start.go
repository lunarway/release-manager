package command

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStart(options *http.Options) *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-manager",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)

			go func() {
				err := http.NewServer(options)
				if err != nil {
					done <- errors.WithMessage(err, "new http server")
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
				log.Errorf("Exited unknown error: %v", err)
				os.Exit(1)
			}
			log.Infof("Program ended")
			return nil
		},
	}

	return command
}

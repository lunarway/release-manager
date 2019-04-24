package command

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/grafana"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type grafanaOptions struct {
	DevAPIKey     string
	DevURL        string
	StagingAPIKey string
	StagingURL    string
	ProdAPIKey    string
	ProdURL       string
}

type configRepoOptions struct {
	ConfigRepo        string
	ArtifactFileName  string
	SSHPrivateKeyPath string
}

func NewStart(grafanaOpts *grafanaOptions, slackAuthToken *string, configRepoOpts *configRepoOptions, httpOpts *http.Options, userMappings map[string]string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-manager",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)
			slackClient, err := slack.NewClient(*slackAuthToken, userMappings)
			if err != nil {
				return err
			}
			grafana := grafana.Service{
				Environments: map[string]grafana.Environment{
					"dev": {
						APIKey:  grafanaOpts.DevAPIKey,
						BaseURL: grafanaOpts.DevURL,
					},
					"staging": {
						APIKey:  grafanaOpts.StagingAPIKey,
						BaseURL: grafanaOpts.StagingURL,
					},
					"prod": {
						APIKey:  grafanaOpts.ProdAPIKey,
						BaseURL: grafanaOpts.ProdURL,
					},
				},
			}
			flowSvc := flow.Service{
				ConfigRepoURL:     configRepoOpts.ConfigRepo,
				ArtifactFileName:  configRepoOpts.ArtifactFileName,
				SSHPrivateKeyPath: configRepoOpts.SSHPrivateKeyPath,
				UserMappings:      userMappings,
				Slack:             slackClient,
				Grafana:           &grafana,
			}
			policySvc := policy.Service{
				ConfigRepoURL:     configRepoOpts.ConfigRepo,
				SSHPrivateKeyPath: configRepoOpts.SSHPrivateKeyPath,
			}
			go func() {
				err := http.NewServer(httpOpts, slackClient, &flowSvc, &policySvc)
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

			err = <-done
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

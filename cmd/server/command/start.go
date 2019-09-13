package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/github"
	"github.com/lunarway/release-manager/internal/grafana"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
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

func NewStart(grafanaOpts *grafanaOptions, slackAuthToken *string, githubAPIToken *string, configRepoOpts *configRepoOptions, httpOpts *http.Options, userMappings *map[string]string) *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-manager",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)
			slackClient, err := slack.NewClient(*slackAuthToken, *userMappings)
			if err != nil {
				return err
			}
			tracer, err := tracing.NewJaeger()
			if err != nil {
				return err
			}
			defer tracer.Close()
			grafanaSvc := grafana.Service{
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
			gitSvc := git.Service{
				Tracer:            tracer,
				SSHPrivateKeyPath: configRepoOpts.SSHPrivateKeyPath,
				ConfigRepoURL:     configRepoOpts.ConfigRepo,
			}
			github := github.Service{
				Token: *githubAPIToken,
			}
			ctx := context.Background()
			close, err := gitSvc.InitMasterRepo(ctx)
			if err != nil {
				return err
			}
			defer close(ctx)
			flowSvc := flow.Service{
				ArtifactFileName: configRepoOpts.ArtifactFileName,
				UserMappings:     *userMappings,
				Slack:            slackClient,
				Git:              &gitSvc,
				Tracer:           tracer,
				// retries for comitting changes into config repo
				// can be required for racing writes
				MaxRetries: 3,
				NotifyReleaseHook: func(ctx context.Context, opts flow.NotifyReleaseOptions) {
					span, ctx := tracer.FromCtx(ctx, "serevier.start NotifyReleaseHook")
					defer span.Finish()
					logger := log.WithFields("service", opts.Service,
						"environment", opts.Environment,
						"namespace", opts.Namespace,
						"artifact-id", opts.Spec.ID,
						"commit-message", opts.Spec.Application.Message,
						"commit-author", opts.Spec.Application.AuthorName,
						"commit-link", opts.Spec.Application.URL,
						"commit-sha", opts.Spec.Application.SHA,
						"releaser", opts.Releaser,
						"type", "release")

					span, _ = tracer.FromCtx(ctx, "notify release channel")
					err := slackClient.NotifySlackReleasesChannel(slack.ReleaseOptions{
						Service:       opts.Service,
						Environment:   opts.Environment,
						ArtifactID:    opts.Spec.ID,
						CommitMessage: opts.Spec.Application.Message,
						CommitAuthor:  opts.Spec.Application.AuthorName,
						CommitLink:    opts.Spec.Application.URL,
						CommitSHA:     opts.Spec.Application.SHA,
						Releaser:      opts.Releaser,
					})
					span.Finish()
					if err != nil {
						logger.Errorf("flow.NotifyReleaseHook: failed to post releases slack message: %v", err)
					}

					span, _ = tracer.FromCtx(ctx, "annotate grafana")
					err = grafanaSvc.Annotate(opts.Environment, grafana.AnnotateRequest{
						What: fmt.Sprintf("Deployment: %s", opts.Service),
						Data: fmt.Sprintf("Author: %s\nMessage: %s\nArtifactID: %s", opts.Spec.Application.AuthorName, opts.Spec.Application.Message, opts.Spec.ID),
						Tags: []string{"deployment", opts.Service},
					})
					span.Finish()
					if err != nil {
						logger.Errorf("flow.NotifyReleaseHook: failed to annotate Grafana: %v", err)
					}

					if strings.ToLower(opts.Spec.Application.Provider) == "github" && *githubAPIToken != "" {
						logger.Infof("Tagging GitHub repository '%s' with '%s' at '%s'", opts.Spec.Application.Name, opts.Environment, opts.Spec.Application.SHA)
						span, _ = tracer.FromCtx(ctx, "tag source repository")
						err = github.TagRepo(ctx, opts.Spec.Application.Name, opts.Environment, opts.Spec.Application.SHA)
						span.Finish()
						if err != nil {
							logger.Errorf("flow.NotifyReleaseHook: failed to tag source repository: %v", err)
						}
					} else {
						logger.Infof("Skipping GitHub repository tagging")
					}

					log.Infof("Release [%s]: %s (%s) by %s, author %s", opts.Environment, opts.Service, opts.Spec.ID, opts.Releaser, opts.Spec.Application.AuthorName)
				},
			}
			policySvc := policy.Service{
				Tracer: tracer,
				Git:    &gitSvc,
				// retries for comitting changes into config repo
				// can be required for racing writes
				MaxRetries: 3,
			}
			go func() {
				err := http.NewServer(httpOpts, slackClient, &flowSvc, &policySvc, &gitSvc, tracer)
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
					log.Infof("received os signal '%s'", sig)
					done <- nil
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

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/amqp"
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

type amqpOptions struct {
	Host                    string
	User                    string
	Password                string
	Port                    int
	VirtualHost             string
	MaxReconnectionAttempts int
	ReconnectionTimeout     time.Duration
	Prefetch                int
	Exchange                string
	Queue                   string
}

type configRepoOptions struct {
	ConfigRepo        string
	ArtifactFileName  string
	SSHPrivateKeyPath string
}

func NewStart(grafanaOpts *grafanaOptions, slackAuthToken *string, githubAPIToken *string, configRepoOpts *configRepoOptions, httpOpts *http.Options, amqpOptions *amqpOptions, userMappings *map[string]string) *cobra.Command {
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
				// TODO: figure out a better way of splitting the consumer and publisher
				// to avoid this chicken and egg issue. It is not a real problem as the
				// consumer is started later on and this we are sure this gets set, it
				// just complicates the flow of the code.
				PublishPromote:           nil,
				PublishReleaseArtifactID: nil,
				PublishReleaseBranch:     nil,
				PublishRollback:          nil,
				// retries for comitting changes into config repo
				// can be required for racing writes
				MaxRetries: 3,
				NotifyReleaseHook: func(ctx context.Context, opts flow.NotifyReleaseOptions) {
					span, ctx := tracer.FromCtx(ctx, "flow.NotifyReleaseHook")
					defer span.Finish()
					logger := log.WithContext(ctx).WithFields("service", opts.Service,
						"environment", opts.Environment,
						"namespace", opts.Namespace,
						"artifact-id", opts.Spec.ID,
						"commit-message", opts.Spec.Application.Message,
						"commit-author", opts.Spec.Application.AuthorName,
						"commit-author-email", opts.Spec.Application.AuthorEmail,
						"commit-comitter", opts.Spec.Application.CommitterName,
						"commit-comitter-email", opts.Spec.Application.CommitterEmail,
						"commit-link", opts.Spec.Application.URL,
						"commit-sha", opts.Spec.Application.SHA,
						"releaser", opts.Releaser,
						"type", "release")

					span, _ = tracer.FromCtx(ctx, "notify release channel")
					err := slackClient.NotifySlackReleasesChannel(ctx, slack.ReleaseOptions{
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
					err = grafanaSvc.Annotate(ctx, opts.Environment, grafana.AnnotateRequest{
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

					logger.Infof("Release [%s]: %s (%s) by %s, author %s", opts.Environment, opts.Service, opts.Spec.ID, opts.Releaser, opts.Spec.Application.AuthorName)
				},
			}

			broker, err := amqp.NewWorker(amqp.Config{
				Connection: amqp.ConnectionConfig{
					Host:        amqpOptions.Host,
					User:        amqpOptions.User,
					Password:    amqpOptions.Password,
					VirtualHost: amqpOptions.VirtualHost,
					Port:        amqpOptions.Port,
				},
				MaxReconnectionAttempts: amqpOptions.MaxReconnectionAttempts,
				ReconnectionTimeout:     amqpOptions.ReconnectionTimeout,
				Exchange:                amqpOptions.Exchange,
				Queue:                   amqpOptions.Queue,
				RoutingKey:              "#",
				Prefetch:                amqpOptions.Prefetch,
				Logger:                  log.With("system", "amqp"),
				Handlers: map[string]func(d []byte) error{
					flow.PromoteEvent{}.Type(): func(d []byte) error {
						var event flow.PromoteEvent
						err := json.Unmarshal(d, &event)
						if err != nil {
							return errors.WithMessage(err, "unmarshal event")
						}
						log.Infof("received promote event: %s", d)
						return flowSvc.ExecPromote(context.Background(), event)
					},
					flow.ReleaseArtifactIDEvent{}.Type(): func(d []byte) error {
						var event flow.ReleaseArtifactIDEvent
						err := json.Unmarshal(d, &event)
						if err != nil {
							return errors.WithMessage(err, "unmarshal event")
						}
						log.Infof("received release artifact id event: %s", d)
						return flowSvc.ExecReleaseArtifactID(context.Background(), event)
					},
					flow.ReleaseBranchEvent{}.Type(): func(d []byte) error {
						var event flow.ReleaseBranchEvent
						err := json.Unmarshal(d, &event)
						if err != nil {
							return errors.WithMessage(err, "unmarshal event")
						}
						log.Infof("received release branch event: %s", d)
						return flowSvc.ExecReleaseBranch(context.Background(), event)
					},
					flow.RollbackEvent{}.Type(): func(d []byte) error {
						var event flow.RollbackEvent
						err := json.Unmarshal(d, &event)
						if err != nil {
							return errors.WithMessage(err, "unmarshal event")
						}
						log.Infof("received rollback event: %s", d)
						return flowSvc.ExecRollback(context.Background(), event)
					},
				},
			})
			if err != nil {
				return err
			}
			flowSvc.PublishPromote = func(ctx context.Context, event flow.PromoteEvent) error {
				return broker.Publish(ctx, event)
			}
			flowSvc.PublishReleaseArtifactID = func(ctx context.Context, event flow.ReleaseArtifactIDEvent) error {
				return broker.Publish(ctx, event)
			}
			flowSvc.PublishReleaseBranch = func(ctx context.Context, event flow.ReleaseBranchEvent) error {
				return broker.Publish(ctx, event)
			}
			flowSvc.PublishRollback = func(ctx context.Context, event flow.RollbackEvent) error {
				return broker.Publish(ctx, event)
			}
			defer func() {
				err := broker.Close()
				if err != nil {
					log.Errorf("Failed to close broker: %v", err)
				}
			}()
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
			go func() {
				err := broker.StartConsumer()
				done <- errors.WithMessage(err, "amqp broker")
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
	return command
}

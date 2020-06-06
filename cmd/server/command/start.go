package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lunarway/release-manager/cmd/server/gpg"
	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/broker/amqp"
	"github.com/lunarway/release-manager/internal/broker/memory"
	"github.com/lunarway/release-manager/internal/fallbackstorage"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/github"
	"github.com/lunarway/release-manager/internal/grafana"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/s3storage"
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

// brokerType represents a configured broker type. It implements pflag.Value to
// support input validation and typesafety.
type brokerType string

const (
	BrokerTypeAMQP   brokerType = "amqp"
	BrokerTypeMemory brokerType = "memory"
)

func (t *brokerType) String() string {
	return string(*t)
}
func (t *brokerType) Set(s string) error {
	switch brokerType(s) {
	case BrokerTypeMemory, BrokerTypeAMQP:
		*t = brokerType(s)
		return nil
	default:
		return errors.New("broker not supported")
	}
}
func (t *brokerType) Type() string {
	return "string"
}

type brokerOptions struct {
	Type   brokerType
	AMQP   amqpOptions
	Memory memoryOptions
}

type amqpOptions struct {
	Host                string
	User                string
	Password            string
	Port                int
	VirtualHost         string
	ReconnectionTimeout time.Duration
	RepublishTimeout    time.Duration
	Prefetch            int
	Exchange            string
	Queue               string
}

type memoryOptions struct {
	QueueSize int
}

type configRepoOptions struct {
	ConfigRepo        string
	ArtifactFileName  string
	SSHPrivateKeyPath string
}

type s3storageOptions struct {
	S3BucketName string
}

type startOptions struct {
	slackAuthToken            *string
	githubAPIToken            *string
	grafana                   *grafanaOptions
	configRepo                *configRepoOptions
	gitConfigOpts             *git.GitConfig
	http                      *http.Options
	broker                    *brokerOptions
	s3storage                 *s3storageOptions
	slackMutes                *slack.MuteOptions
	gpgKeyPaths               *[]string
	userMappings              *map[string]string
	branchRestrictionPolicies *[]policy.BranchRestriction
}

func NewStart(startOptions *startOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "start",
		Short: "start the release-manager",
		RunE: func(c *cobra.Command, args []string) error {
			done := make(chan error, 1)
			slackClient, err := slack.NewMuteableClient(*startOptions.slackAuthToken, *startOptions.userMappings, *startOptions.slackMutes)
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
						APIKey:  startOptions.grafana.DevAPIKey,
						BaseURL: startOptions.grafana.DevURL,
					},
					"staging": {
						APIKey:  startOptions.grafana.StagingAPIKey,
						BaseURL: startOptions.grafana.StagingURL,
					},
					"prod": {
						APIKey:  startOptions.grafana.ProdAPIKey,
						BaseURL: startOptions.grafana.ProdURL,
					},
				},
			}
			// Import GPG Keys
			if startOptions.gitConfigOpts.SigningKey != "" {
				if len(*startOptions.gpgKeyPaths) < 1 {
					return errors.New("gpg signing key provided, but no import paths specified")
				}
				for _, p := range *startOptions.gpgKeyPaths {
					// lets just use flux' implementation on how to load keys
					keyfiles, err := gpg.ImportKeys(p, false)
					if err != nil {
						return fmt.Errorf("failed to import GPG key(s) from %s", p)
					}
					if keyfiles != nil {
						log.Infof("imported GPG key(s) from %s files %v", p, keyfiles)
					}
				}
			}
			gitSvc := git.Service{
				Tracer:            tracer,
				SSHPrivateKeyPath: startOptions.configRepo.SSHPrivateKeyPath,
				ConfigRepoURL:     startOptions.configRepo.ConfigRepo,
				Config:            startOptions.gitConfigOpts,
				ArtifactFileName:  startOptions.configRepo.ArtifactFileName,
			}

			var s3storageSvc *s3storage.Service
			if startOptions.s3storage.S3BucketName != "" {
				s3storageSvc, err = s3storage.New(startOptions.s3storage.S3BucketName, tracer)
				if err != nil {
					return err
				}
			}

			github := github.Service{
				Token: *startOptions.githubAPIToken,
			}
			ctx := context.Background()
			close, err := gitSvc.InitMasterRepo(ctx)
			if err != nil {
				return err
			}
			defer close(ctx)
			policySvc := policy.Service{
				Tracer: tracer,
				Git:    &gitSvc,
				// retries for comitting changes into config repo
				// can be required for racing writes
				MaxRetries:                      3,
				GlobalBranchRestrictionPolicies: *startOptions.branchRestrictionPolicies,
			}

			var storage flow.ArtifactReadStorage = &gitSvc
			if s3storageSvc != nil {
				storage = fallbackstorage.New(s3storageSvc, storage, tracer)
			}

			flowSvc := flow.Service{
				ArtifactFileName: startOptions.configRepo.ArtifactFileName,
				UserMappings:     *startOptions.userMappings,
				Slack:            slackClient,
				Git:              &gitSvc,
				CanRelease:       policySvc.CanRelease,
				Storage:          storage,
				Policy:           &policySvc,
				Tracer:           tracer,
				// TODO: figure out a better way of splitting the consumer and publisher
				// to avoid this chicken and egg issue. It is not a real problem as the
				// consumer is started later on and this we are sure this gets set, it
				// just complicates the flow of the code.
				PublishPromote:           nil,
				PublishReleaseArtifactID: nil,
				PublishReleaseBranch:     nil,
				PublishRollback:          nil,
				PublishNewArtifact:       nil,
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

					releaseOptions := slack.ReleaseOptions{
						Service:           opts.Service,
						Environment:       opts.Environment,
						ArtifactID:        opts.Spec.ID,
						CommitMessage:     opts.Spec.Application.Message,
						CommitAuthor:      opts.Spec.Application.AuthorName,
						CommitAuthorEmail: opts.Spec.Application.AuthorEmail,
						CommitLink:        opts.Spec.Application.URL,
						CommitSHA:         opts.Spec.Application.SHA,
						Releaser:          opts.Releaser,
					}
					span, _ = tracer.FromCtx(ctx, "notify release channel")
					err := slackClient.NotifySlackReleasesChannel(ctx, releaseOptions)
					span.Finish()
					if err != nil {
						logger.Errorf("flow.NotifyReleaseHook: failed to post releases slack message: %v", err)
					}

					span, _ = tracer.FromCtx(ctx, "notify author")
					err = slackClient.NotifyAuthorEventProcessed(ctx, releaseOptions)
					span.Finish()
					if err != nil {
						logger.Errorf("flow.NotifyReleaseHook: failed to post slack event processed message to author: %v", err)
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

					if strings.ToLower(opts.Spec.Application.Provider) == "github" && *startOptions.githubAPIToken != "" {
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

			eventHandlers := map[string]func([]byte) error{
				flow.PromoteEvent{}.Type(): func(d []byte) error {
					var event flow.PromoteEvent
					err := event.Unmarshal(d)
					if err != nil {
						return errors.WithMessage(err, "unmarshal event")
					}
					return flowSvc.ExecPromote(context.Background(), event)
				},
				flow.ReleaseArtifactIDEvent{}.Type(): func(d []byte) error {
					var event flow.ReleaseArtifactIDEvent
					err := event.Unmarshal(d)
					if err != nil {
						return errors.WithMessage(err, "unmarshal event")
					}
					return flowSvc.ExecReleaseArtifactID(context.Background(), event)
				},
				flow.ReleaseBranchEvent{}.Type(): func(d []byte) error {
					var event flow.ReleaseBranchEvent
					err := event.Unmarshal(d)
					if err != nil {
						return errors.WithMessage(err, "unmarshal event")
					}
					return flowSvc.ExecReleaseBranch(context.Background(), event)
				},
				flow.RollbackEvent{}.Type(): func(d []byte) error {
					var event flow.RollbackEvent
					err := event.Unmarshal(d)
					if err != nil {
						return errors.WithMessage(err, "unmarshal event")
					}
					return flowSvc.ExecRollback(context.Background(), event)
				},
				flow.NewArtifactEvent{}.Type(): func(d []byte) error {
					var event flow.NewArtifactEvent
					err := event.Unmarshal(d)
					if err != nil {
						return errors.WithMessage(err, "unmarshal event")
					}
					return flowSvc.ExecNewArtifact(context.Background(), event)
				},
			}

			errorHandler := func(msgType string, msgBody []byte, err error) {
				var event flow.GenericEvent
				unmarshalErr := event.Unmarshal(msgBody)
				if unmarshalErr != nil {
					log.Errorf("errorhandling could not unmarshal event of type %s to generic event with error: %s", msgType, unmarshalErr)
					return
				}
				slackErr := slackClient.NotifyReleaseManagerError(ctx, msgType, event.Service, event.Environment, event.Branch, event.Namespace, event.Actor.Email, err)
				if slackErr != nil {
					log.Errorf("slack notification failed with error %s", slackErr)
				}
			}

			brokerImpl, err := getBroker(startOptions.broker)
			if err != nil {
				return errors.WithMessage(err, "setup broker")
			}

			flowSvc.PublishPromote = func(ctx context.Context, event flow.PromoteEvent) error {
				return brokerImpl.Publish(ctx, &event)
			}
			flowSvc.PublishReleaseArtifactID = func(ctx context.Context, event flow.ReleaseArtifactIDEvent) error {
				return brokerImpl.Publish(ctx, &event)
			}
			flowSvc.PublishReleaseBranch = func(ctx context.Context, event flow.ReleaseBranchEvent) error {
				return brokerImpl.Publish(ctx, &event)
			}
			flowSvc.PublishRollback = func(ctx context.Context, event flow.RollbackEvent) error {
				return brokerImpl.Publish(ctx, &event)
			}
			flowSvc.PublishNewArtifact = func(ctx context.Context, event flow.NewArtifactEvent) error {
				return brokerImpl.Publish(ctx, &event)
			}
			defer func() {
				err := brokerImpl.Close()
				if err != nil {
					log.Errorf("Failed to close broker: %v", err)
				}
			}()
			go func() {
				err := http.NewServer(startOptions.http, slackClient, &flowSvc, &policySvc, &gitSvc, s3storageSvc, tracer)
				if err != nil {
					done <- errors.WithMessage(err, "new http server")
					return
				}
			}()
			go func() {
				err := brokerImpl.StartConsumer(eventHandlers, errorHandler)
				done <- errors.WithMessage(err, "broker")
			}()

			if s3storageSvc != nil {
				sqsHandler := func(msg string) error {
					var s3event s3storage.S3Event
					err := s3event.Unmarshal([]byte(msg))
					if err != nil {
						return errors.WithMessage(err, "unmarshal S3Event")
					}

					if len(s3event.Records) == 0 {
						log.With("sqsMessage", msg).Infof("Skipping SQS message because it does not look like S3 notification")
						return nil
					}

					for _, record := range s3event.Records {
						parts := strings.Split(record.S3.Object.Key, "/")
						if len(parts) != 2 {
							log.With("s3event", s3event).Infof("Got s3 object creation event on %s which can't be parsed to service and artifact id", record.S3.Object.Key)
							continue
						}
						flowSvc.NewArtifact(ctx, parts[0], parts[1])
					}
					return nil
				}

				err = s3storageSvc.InitializeBucket()
				if err != nil {
					return err
				}
				err = s3storageSvc.InitializeSQS(sqsHandler)
				if err != nil {
					return err
				}
				defer func() {
					err := s3storageSvc.Close()
					if err != nil {
						log.Errorf("Failed to close s3 storage: %v", err)
					}
				}()
			}

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

func getBroker(c *brokerOptions) (broker.Broker, error) {
	switch c.Type {
	case BrokerTypeAMQP:
		amqpOptions := c.AMQP
		log.Info("Using an AMQP broker")
		return amqp.NewWorker(amqp.Config{
			Connection: amqp.ConnectionConfig{
				Host:        amqpOptions.Host,
				User:        amqpOptions.User,
				Password:    amqpOptions.Password,
				VirtualHost: amqpOptions.VirtualHost,
				Port:        amqpOptions.Port,
			},
			ReconnectionTimeout: amqpOptions.ReconnectionTimeout,
			RepublishTimeout:    amqpOptions.RepublishTimeout,
			Exchange:            amqpOptions.Exchange,
			Queue:               amqpOptions.Queue,
			RoutingKey:          "#",
			Prefetch:            amqpOptions.Prefetch,
			Logger:              log.With("system", "amqp"),
		})
	case BrokerTypeMemory:
		queueSize := c.Memory.QueueSize
		if queueSize <= 0 {
			queueSize = 5
		}
		log.Infof("Using an in-memory broker with queue size %d", queueSize)
		return memory.New(log.With("system", "memory"), queueSize), nil
	default:
		// this should never happen as the flags are validated against available
		// values
		return nil, errors.New("no broker selected")
	}
}

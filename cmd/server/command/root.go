package command

import (
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewRoot returns a new instance of a hamctl command.
func NewRoot(version string) (*cobra.Command, error) {
	var brokerOpts brokerOptions
	var httpOpts http.Options
	var grafanaOpts grafanaOptions
	var slackAuthToken string
	var githubAPIToken string
	var configRepoOpts configRepoOptions
	var gitConfigOpts git.GitConfig
	var gpgKeyPaths []string
	var users []string
	var userMappings map[string]string
	var branchRestrictionsList []string
	var branchRestrictions []policy.BranchRestriction
	var logConfiguration *log.Configuration
	var slackMuteOpts slack.MuteOptions
	var s3storageOpts s3storageOptions
	var emailSuffix string

	var command = &cobra.Command{
		Use:   "server",
		Short: "server",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			if len(users) == 0 {
				userMappingString := os.Getenv("USER_MAPPINGS")
				users = strings.Split(userMappingString, ",")
			}
			var err error
			userMappings, err = slack.ParseUserMappings(users)
			if err != nil {
				return errors.WithMessage(err, "user mappings")
			}
			branchRestrictions, err = parseBranchRestrictions(branchRestrictionsList)
			if err != nil {
				return errors.WithMessage(err, "branch restrictions")
			}

			logConfiguration.ParseFromEnvironmnet()
			log.Init(logConfiguration)
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(
		NewStart(&startOptions{
			grafana:                   &grafanaOpts,
			slackAuthToken:            &slackAuthToken,
			githubAPIToken:            &githubAPIToken,
			configRepo:                &configRepoOpts,
			gitConfigOpts:             &gitConfigOpts,
			s3storage:                 &s3storageOpts,
			http:                      &httpOpts,
			gpgKeyPaths:               &gpgKeyPaths,
			broker:                    &brokerOpts,
			slackMutes:                &slackMuteOpts,
			userMappings:              &userMappings,
			branchRestrictionPolicies: &branchRestrictions,
			emailSuffix:               &emailSuffix,
		}),
		NewVersion(version),
	)
	command.PersistentFlags().IntVar(&httpOpts.Port, "http-port", 8080, "port of the http server")
	command.PersistentFlags().DurationVar(&httpOpts.Timeout, "timeout", 20*time.Second, "HTTP server timeout for incomming requests")
	command.PersistentFlags().StringVar(&httpOpts.HamCtlAuthToken, "hamctl-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "hamctl authentication token")
	command.PersistentFlags().StringVar(&httpOpts.ArtifactAuthToken, "artifact-auth-token", os.Getenv("ARTIFACT_AUTH_TOKEN"), "artifact authentication token")
	command.PersistentFlags().StringVar(&httpOpts.DaemonAuthToken, "daemon-auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "daemon webhook authentication token")
	command.PersistentFlags().StringVar(&configRepoOpts.ConfigRepo, "config-repo", os.Getenv("CONFIG_REPO"), "ssh url for the git config repository")
	command.PersistentFlags().StringVar(&configRepoOpts.ArtifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	command.PersistentFlags().StringVar(&configRepoOpts.SSHPrivateKeyPath, "ssh-private-key", "/etc/release-manager/ssh/identity", "ssh-private-key for the config repo")
	command.PersistentFlags().StringVar(&httpOpts.GithubWebhookSecret, "github-webhook-secret", os.Getenv("GITHUB_WEBHOOK_SECRET"), "github webhook secret")
	command.PersistentFlags().StringVar(&githubAPIToken, "github-api-token", os.Getenv("GITHUB_API_TOKEN"), "github api token for tagging releases")
	command.PersistentFlags().StringVar(&slackAuthToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token to be used to communicate with the slack api")
	command.PersistentFlags().StringVar(&grafanaOpts.DevAPIKey, "grafana-api-key-dev", os.Getenv("GRAFANA_DEV_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&grafanaOpts.StagingAPIKey, "grafana-api-key-staging", os.Getenv("GRAFANA_STAGING_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&grafanaOpts.ProdAPIKey, "grafana-api-key-prod", os.Getenv("GRAFANA_PROD_API_KEY"), "api key to be used to annotate in prod")
	command.PersistentFlags().StringVar(&grafanaOpts.DevURL, "grafana-dev-url", os.Getenv("GRAFANA_DEV_URL"), "grafana dev url")
	command.PersistentFlags().StringVar(&grafanaOpts.StagingURL, "grafana-staging-url", os.Getenv("GRAFANA_STAGING_URL"), "grafana staging url")
	command.PersistentFlags().StringVar(&grafanaOpts.ProdURL, "grafana-prod-url", os.Getenv("GRAFANA_PROD_URL"), "grafana prod url")
	command.PersistentFlags().StringVar(&emailSuffix, "email-suffix", "", "company email suffix to expect. E.g.: '@example.com'")
	command.PersistentFlags().StringSliceVar(&users, "user-mappings", []string{}, "user mappings between emails used by Git and Slack, key-value pair: <email>=<slack-email>")
	command.PersistentFlags().StringSliceVar(&branchRestrictionsList, "policy-branch-restrictions", []string{}, "branch restriction policies applied to all releases, key-value pair: <environment>=<branch-regex>")
	command.PersistentFlags().StringSliceVar(&gpgKeyPaths, "git-gpg-key-import-paths", []string{}, "a list of paths for signing keys to import to gpg")

	registerBrokerFlags(command, &brokerOpts)
	registerSlackNotificationFlags(command, &slackMuteOpts)
	registerGitFlags(command, &gitConfigOpts)
	registerS3Flags(command, &s3storageOpts)
	logConfiguration = log.RegisterFlags(command)

	return command, nil
}

func registerSlackNotificationFlags(cmd *cobra.Command, opts *slack.MuteOptions) {
	cmd.PersistentFlags().BoolVar(&opts.Kubernetes, "mute-slack-notification-k8s", false, "Enable/disable k8s slack notifications")
	cmd.PersistentFlags().BoolVar(&opts.Policy, "mute-slack-notification-policy", false, "Enable/disable policies slack notifications")
	cmd.PersistentFlags().BoolVar(&opts.ReleaseProcessed, "mute-slack-notification-release-processed", false, "Enable/disable release processed slack notifications")
}

func registerBrokerFlags(cmd *cobra.Command, c *brokerOptions) {
	cmd.PersistentFlags().Var(&c.Type, "broker-type", "configure what broker to use. Available values are \"memory\" and \"amqp\"")

	// in-memory options
	cmd.PersistentFlags().IntVar(&c.Memory.QueueSize, "memory-queue-size", 5, "in-memory queue size")

	// amqp options
	cmd.PersistentFlags().StringVar(&c.AMQP.Host, "amqp-host", "localhost", "AMQP host URL")
	cmd.PersistentFlags().IntVar(&c.AMQP.Port, "amqp-port", 5672, "AMQP host port")
	cmd.PersistentFlags().StringVar(&c.AMQP.User, "amqp-user", "", "AMQP user name")
	cmd.PersistentFlags().StringVar(&c.AMQP.Password, "amqp-password", "", "AMQP password")
	cmd.PersistentFlags().StringVar(&c.AMQP.VirtualHost, "amqp-virtualhost", "/", "AMQP virtual host")
	cmd.PersistentFlags().DurationVar(&c.AMQP.ReconnectionTimeout, "amqp-reconnection-timeouts", 5*time.Second, "AMQP reconnection attempt timeout")
	cmd.PersistentFlags().DurationVar(&c.AMQP.ConnectionTimeout, "amqp-connection-timeouts", 10*time.Second, "AMQP dial connection timeout")
	cmd.PersistentFlags().DurationVar(&c.AMQP.InitTimeout, "amqp-init-timeouts", 10*time.Second, "AMQP initialization timeout")
	cmd.PersistentFlags().IntVar(&c.AMQP.Prefetch, "amqp-prefetch", 1, "AMQP queue prefetch")
	cmd.PersistentFlags().StringVar(&c.AMQP.Exchange, "amqp-exchange", "release-manager", "AMQP exchange")
	cmd.PersistentFlags().StringVar(&c.AMQP.Queue, "amqp-queue", "release-manager", "AMQP queue")
}

func registerGitFlags(cmd *cobra.Command, opts *git.GitConfig) {
	cmd.PersistentFlags().StringVar(&opts.User, "git-user", "HamAstrochimp", "the user that all commits will be committed with.")
	cmd.PersistentFlags().StringVar(&opts.Email, "git-email", "operations@lunar.app", "the email that all commits will be committed with.")
	cmd.PersistentFlags().StringVar(&opts.SigningKey, "git-signing-key", "", "the signingkey which all commits will be signed with. The path to the key has to be provided.")
}

func registerS3Flags(cmd *cobra.Command, opts *s3storageOptions) {
	cmd.PersistentFlags().StringVar(&opts.S3BucketName, "s3-artifact-storage-bucket-name", "", "the S3 bucket to store artifacts in.")
}

// parseBranchRestrictions pases a slice of key-value pairs formatted as
// <environment>=<branchRegex>. It will return an error if the format is invalid
// and if multiple retrictions conflict, ie. multiple restrictions on one
// environment.
func parseBranchRestrictions(list []string) ([]policy.BranchRestriction, error) {
	// use a map to detect conflicting restrictions on environment
	m := make(map[string]policy.BranchRestriction)
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		s := strings.Split(item, "=")
		if len(s) != 2 {
			return nil, errors.Errorf("invalid format '%s'", item)
		}
		environment := strings.TrimSpace(s[0])
		branchRegex := strings.TrimSpace(s[1])
		_, exist := m[environment]
		if exist {
			return nil, errors.Errorf("conflicting mappings for %s", environment)
		}
		if environment == "" || branchRegex == "" {
			return nil, errors.Errorf("invalid mapping '%s'", item)
		}
		m[environment] = policy.BranchRestriction{
			ID:          "", // effectively protects against attempts to delete the policy.
			Environment: environment,
			BranchRegex: branchRegex,
		}
	}

	// flatten map to a slice
	var restrictions []policy.BranchRestriction
	for _, r := range m {
		restrictions = append(restrictions, r)
	}
	log.Infof("Parsed %d global branch restriction policies", len(restrictions))
	return restrictions, nil
}

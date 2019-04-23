package command

import (
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/spf13/cobra"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand() (*cobra.Command, error) {
	var httpOpts http.Options
	var grafanaOpts grafanaOptions
	var slackAuthToken string
	var configRepoOpts configRepoOptions
	var users []string
	var userMappings map[string]string

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
				return err
			}
			httpOpts.UserMappings = userMappings
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewStart(&grafanaOpts, &slackAuthToken, &configRepoOpts, &httpOpts, userMappings))
	command.PersistentFlags().IntVar(&httpOpts.Port, "http-port", 8080, "port of the http server")
	command.PersistentFlags().DurationVar(&httpOpts.Timeout, "timeout", 20*time.Second, "HTTP server timeout for incomming requests")
	command.PersistentFlags().StringVar(&httpOpts.HamCtlAuthToken, "hamctl-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "hamctl authentication token")
	command.PersistentFlags().StringVar(&httpOpts.DaemonAuthToken, "daemon-auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "daemon webhook authentication token")
	command.PersistentFlags().StringVar(&configRepoOpts.ConfigRepo, "config-repo", os.Getenv("CONFIG_REPO"), "ssh url for the git config repository")
	command.PersistentFlags().StringVar(&configRepoOpts.ArtifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	command.PersistentFlags().StringVar(&configRepoOpts.SSHPrivateKeyPath, "ssh-private-key", "/etc/release-manager/ssh/identity", "ssh-private-key for the config repo")
	command.PersistentFlags().StringVar(&httpOpts.GithubWebhookSecret, "github-webhook-secret", os.Getenv("GITHUB_WEBHOOK_SECRET"), "github webhook secret")
	command.PersistentFlags().StringVar(&slackAuthToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token to be used to communicate with the slack api")
	command.PersistentFlags().StringVar(&grafanaOpts.DevAPIKey, "grafana-api-key-dev", os.Getenv("GRAFANA_DEV_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&grafanaOpts.StagingAPIKey, "grafana-api-key-staging", os.Getenv("GRAFANA_STAGING_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&grafanaOpts.ProdAPIKey, "grafana-api-key-prod", os.Getenv("GRAFANA_PROD_API_KEY"), "api key to be used to annotate in prod")
	command.PersistentFlags().StringVar(&grafanaOpts.DevURL, "grafana-dev-url", os.Getenv("GRAFANA_DEV_URL"), "grafana dev url")
	command.PersistentFlags().StringVar(&grafanaOpts.StagingURL, "grafana-staging-url", os.Getenv("GRAFANA_STAGING_URL"), "grafana staging url")
	command.PersistentFlags().StringVar(&grafanaOpts.ProdURL, "grafana-prod-url", os.Getenv("GRAFANA_PROD_URL"), "grafana prod url")
	command.PersistentFlags().StringSliceVar(&users, "user-mappings", []string{}, "user mappings between emails used by Git and Slack, key-value pair: <email>=<slack-email>")

	return command, nil
}

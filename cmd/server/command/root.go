package command

import (
	"os"
	"strings"
	"time"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/spf13/cobra"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand() (*cobra.Command, error) {
	var options http.Options
	var users []string
	var command = &cobra.Command{
		Use:   "server",
		Short: "server",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			if len(users) == 0 {
				userMappingString := os.Getenv("USER_MAPPINGS")
				users = strings.Split(userMappingString, ",")
			}
			m := make(map[string]string)
			for _, u := range users {
				s := strings.Split(u, "=")
				m[s[0]] = s[1]
			}
			options.UserMappings = m
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewStart(&options))
	command.PersistentFlags().IntVar(&options.Port, "http-port", 8080, "port of the http server")
	command.PersistentFlags().DurationVar(&options.Timeout, "timeout", 20*time.Second, "HTTP server timeout for incomming requests")
	command.PersistentFlags().StringVar(&options.HamCtlAuthToken, "hamctl-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "hamctl authentication token")
	command.PersistentFlags().StringVar(&options.DaemonAuthToken, "daemon-auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "daemon webhook authentication token")
	command.PersistentFlags().StringVar(&options.ConfigRepo, "config-repo", os.Getenv("CONFIG_REPO"), "ssh url for the git config repository")
	command.PersistentFlags().StringVar(&options.ArtifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	command.PersistentFlags().StringVar(&options.SSHPrivateKeyPath, "ssh-private-key", "/etc/release-manager/ssh/identity", "ssh-private-key for the config repo")
	command.PersistentFlags().StringVar(&options.GithubWebhookSecret, "github-webhook-secret", os.Getenv("GITHUB_WEBHOOK_SECRET"), "github webhook secret")
	command.PersistentFlags().StringVar(&options.SlackAuthToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token to be used to communicate with the slack api")
	command.PersistentFlags().StringVar(&options.GrafanaDevAPIKey, "grafana-api-key-dev", os.Getenv("GRAFANA_DEV_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&options.GrafanaStagingAPIKey, "grafana-api-key-staging", os.Getenv("GRAFANA_STAGING_API_KEY"), "api key to be used to annotate in dev")
	command.PersistentFlags().StringVar(&options.GrafanaProdAPIKey, "grafana-api-key-prod", os.Getenv("GRAFANA_PROD_API_KEY"), "api key to be used to annotate in prod")
	command.PersistentFlags().StringVar(&options.GrafanaDevUrl, "grafana-dev-url", os.Getenv("GRAFANA_DEV_URL"), "grafana dev url")
	command.PersistentFlags().StringVar(&options.GrafanaStagingUrl, "grafana-staging-url", os.Getenv("GRAFANA_STAGING_URL"), "grafana staging url")
	command.PersistentFlags().StringVar(&options.GrafanaProdUrl, "grafana-prod-url", os.Getenv("GRAFANA_PROD_URL"), "grafana prod url")
	command.PersistentFlags().StringSliceVar(&users, "user-mappings", []string{}, "user mappings between to emails used by Slack, key-value pair: <email>=<slack-email>")

	return command, nil
}

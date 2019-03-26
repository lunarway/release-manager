package command

import (
	"os"
	"time"

	"github.com/lunarway/release-manager/cmd/server/http"
	"github.com/spf13/cobra"
)

// NewCommand returns a new instance of a hamctl command.
func NewCommand() *cobra.Command {
	var options http.Options
	var command = &cobra.Command{
		Use:   "server",
		Short: "server",
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
		},
	}
	command.AddCommand(NewStart(&options))
	command.PersistentFlags().IntVar(&options.Port, "http-port", 8080, "port of the http server")
	command.PersistentFlags().DurationVar(&options.Timeout, "timeout", 20*time.Second, "HTTP server timeout for incomming requests")
	command.PersistentFlags().StringVar(&options.HamCtlAuthToken, "hamctl-auth-token", os.Getenv("HAMCTL_AUTH_TOKEN"), "hamctl authentication token")
	command.PersistentFlags().StringVar(&options.DaemonAuthToken, "daemon-auth-token", os.Getenv("DAEMON_AUTH_TOKEN"), "daemon webhook authentication token")
	command.PersistentFlags().StringVar(&options.ConfigRepo, "config-repo", "git@github.com:lunarway/k8s-cluster-config.git", "ssh url for the git config repository")
	command.PersistentFlags().StringVar(&options.ArtifactFileName, "artifact-filename", "artifact.json", "the filename of the artifact to be used")
	command.PersistentFlags().StringVar(&options.SSHPrivateKeyPath, "ssh-private-key", "/etc/release-manager/ssh/identity", "ssh-private-key for the config repo")
	command.PersistentFlags().StringVar(&options.GithubWebhookSecret, "github-webhook-secret", os.Getenv("GITHUB_WEBHOOK_SECRET"), "github webhook secret")
	command.PersistentFlags().StringVar(&options.SlackAuthToken, "slack-token", os.Getenv("SLACK_TOKEN"), "token to be used to communicate with the slack api")

	return command
}

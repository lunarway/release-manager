package command

import (
	"github.com/lunarway/release-manager/internal/http"
	"github.com/spf13/cobra"
)

func Login(authenticator http.UserAuthenticator) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: `Log into the configured IdP`,
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			return authenticator.Login()
		},
	}
}

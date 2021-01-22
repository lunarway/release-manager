package slack_test

import (
	"testing"

	"github.com/lunarway/release-manager/internal/log"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

// TestNewMutableClient_mapping tests that a mutable client handles email
// mappings correctly. This test is added to fix a bug introduced in
// 73d80a895f35015e30975ef9b75fcaea14493926 where domain verification was added
// to slack messages. The feature was only implemented in the default Client
// (non mutable) which made the server unable to send messages to emails to
// non-slack emails.
func TestNewMutableClient_mapping(t *testing.T) {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})
	var (
		email       = "outside@mail.com"
		emailDomain = "corp.com"
		mapping     = map[string]string{
			"outside@mail.com": "me@corp.com",
		}
		slackUser = &slack.User{
			ID: "user-id",
		}
	)

	slackClient := intslack.MockSlackClient{}
	slackClient.Test(t)
	slackClient.On("PostMessage", slackUser.ID, mock.Anything, mock.Anything).Return("", "", nil)
	// return error when looking up non-corp email
	slackClient.On("GetUserByEmailContext", mock.Anything, email).Return(&slack.User{}, errors.New("users_not_found"))
	// return slack user on anything else
	slackClient.On("GetUserByEmailContext", mock.Anything, mock.Anything).Return(slackUser, nil)

	client, err := intslack.NewMuteableClient(&slackClient, mapping, emailDomain, intslack.MuteOptions{})
	if !assert.NoError(t, err, "unexpected instantiation error") {
		return
	}

	_, _, err = client.PostSlackBuildStarted(email, "title", "titleLink", "text", "color")
	if !assert.NoError(t, err, "unexpected instantiation error") {
		return
	}
}

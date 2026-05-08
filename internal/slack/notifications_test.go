package slack_test

import (
	"context"
	"testing"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	intslack "github.com/lunarway/release-manager/internal/slack"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

func initTestLogger() {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})
}

func TestNotifyRelease_postsSquadReleaseChannelsBestEffort(t *testing.T) {
	initTestLogger()

	slackClient := intslack.MockSlackClient{}
	slackClient.Test(t)
	slackUser := &slack.User{ID: "user-id"}

	slackClient.On("PostMessageContext", mock.Anything, "#releases-dev", mock.Anything, mock.Anything).
		Return("", "", nil).Once()
	slackClient.On("PostMessageContext", mock.Anything, "#squad-alpha-releases-dev", mock.Anything, mock.Anything).
		Return("", "", errors.New("channel_not_found")).Once()
	slackClient.On("PostMessageContext", mock.Anything, "#squad-beta-releases-dev", mock.Anything, mock.Anything).
		Return("", "", nil).Once()
	slackClient.On("GetUserByEmailContext", mock.Anything, "author@corp.com").
		Return(slackUser, nil).Once()
	slackClient.On("PostMessageContext", mock.Anything, slackUser.ID, mock.Anything, mock.Anything).
		Return("", "", nil).Once()

	client, err := intslack.NewClient(&slackClient, nil, "corp.com")
	if !assert.NoError(t, err, "unexpected instantiation error") {
		return
	}

	client.NotifyRelease(context.Background(), intslack.ReleaseOptions{
		Service:           "release-manager",
		ArtifactID:        "artifact-1",
		CommitMessage:     "ship it",
		CommitAuthor:      "Author",
		CommitAuthorEmail: "author@corp.com",
		CommitLink:        "https://example.com",
		Environment:       "dev",
		Releaser:          "Releaser",
		Squads:            []string{"beta", "alpha", "", "alpha"},
	})

	slackClient.AssertExpectations(t)
}

func TestNotifyK8SDeployEvent_postsSquadReleaseChannelBestEffort(t *testing.T) {
	initTestLogger()

	t.Run("squad channel failure is ignored", func(t *testing.T) {
		slackClient := intslack.MockSlackClient{}
		slackClient.Test(t)
		slackUser := &slack.User{ID: "user-id"}

		slackClient.On("GetUserByEmailContext", mock.Anything, "author@corp.com").
			Return(slackUser, nil).Once()
		slackClient.On("PostMessageContext", mock.Anything, slackUser.ID, mock.Anything, mock.Anything).
			Return("", "", nil).Once()
		slackClient.On("PostMessageContext", mock.Anything, "#squad-sentinel-releases-prod", mock.Anything, mock.Anything).
			Return("", "", errors.New("channel_not_found")).Once()

		client, err := intslack.NewClient(&slackClient, nil, "corp.com")
		if !assert.NoError(t, err, "unexpected instantiation error") {
			return
		}

		err = client.NotifyK8SDeployEvent(context.Background(), intslack.NotifyK8sDeployOptions{
			AuthorEmail:   "author@corp.com",
			Environment:   "prod",
			Name:          "deploy",
			AvailablePods: 2,
			DesiredPods:   2,
			ResourceType:  "Deployment",
			ArtifactID:    "artifact-1",
			Squad:         "sentinel",
		})

		assert.NoError(t, err)
		slackClient.AssertExpectations(t)
	})

	t.Run("primary notification error is still returned after squad fan out", func(t *testing.T) {
		slackClient := intslack.MockSlackClient{}
		slackClient.Test(t)

		slackClient.On("GetUserByEmailContext", mock.Anything, "author@corp.com").
			Return(&slack.User{}, errors.New("users_not_found")).Once()
		slackClient.On("PostMessageContext", mock.Anything, "#squad-sentinel-releases-prod", mock.Anything, mock.Anything).
			Return("", "", nil).Once()

		client, err := intslack.NewClient(&slackClient, nil, "corp.com")
		if !assert.NoError(t, err, "unexpected instantiation error") {
			return
		}

		err = client.NotifyK8SDeployEvent(context.Background(), intslack.NotifyK8sDeployOptions{
			AuthorEmail:   "author@corp.com",
			Environment:   "prod",
			Name:          "deploy",
			AvailablePods: 2,
			DesiredPods:   2,
			ResourceType:  "Deployment",
			ArtifactID:    "artifact-1",
			Squad:         "sentinel",
		})

		assert.EqualError(t, err, "users_not_found")
		slackClient.AssertExpectations(t)
	})
}

func TestNotifyK8SPodErrorEvent_postsSquadReleaseChannelBestEffort(t *testing.T) {
	initTestLogger()

	slackClient := intslack.MockSlackClient{}
	slackClient.Test(t)
	slackUser := &slack.User{ID: "user-id"}

	slackClient.On("GetUserByEmailContext", mock.Anything, "author@corp.com").
		Return(slackUser, nil).Once()
	slackClient.On("PostMessageContext", mock.Anything, slackUser.ID, mock.Anything, mock.Anything).
		Return("", "", nil).Once()
	slackClient.On("PostMessageContext", mock.Anything, "#squad-sentinel-alerts", mock.Anything, mock.Anything).
		Return("", "", nil).Once()
	slackClient.On("PostMessageContext", mock.Anything, "#squad-sentinel-releases-dev", mock.Anything, mock.Anything).
		Return("", "", errors.New("channel_not_found")).Once()

	client, err := intslack.NewClient(&slackClient, nil, "corp.com")
	if !assert.NoError(t, err, "unexpected instantiation error") {
		return
	}

	err = client.NotifyK8SPodErrorEvent(context.Background(), &httpinternal.PodErrorEvent{
		PodName:     "pod-1",
		AuthorEmail: "author@corp.com",
		Environment: "dev",
		ArtifactID:  "artifact-1",
		Squad:       "sentinel",
		AlertSquad:  "#squad-sentinel-alerts",
	})

	assert.NoError(t, err)
	slackClient.AssertExpectations(t)
}

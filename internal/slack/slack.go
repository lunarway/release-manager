package slack

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/spec"
	"github.com/nlopes/slack"
)

type Client struct {
	client *slack.Client
}

func NewClient(token string) (*Client, error) {
	slackClient := slack.New(token)
	client := Client{
		client: slackClient,
	}
	return &client, nil
}

func (c *Client) GetSlackIdByEmail(email string) (string, error) {
	user, err := c.client.GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (c *Client) PostPrivateMessage(userID, env, service string, artifact spec.Spec, request *http.StatusNotifyRequest) error {
	asUser := slack.MsgOptionAsUser(true)
	podField := slack.AttachmentField{
		Title: "Pod",
		Value: request.PodName,
		Short: true,
	}
	statusField := slack.AttachmentField{
		Title: "Status",
		Value: request.Status,
		Short: true,
	}
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:  fmt.Sprintf("Pod status changed for %s in %s (artifact id: %s)", service, env, artifact.ID),
		Fields: []slack.AttachmentField{podField, statusField},
	})

	_, _, err := c.client.PostMessage(userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

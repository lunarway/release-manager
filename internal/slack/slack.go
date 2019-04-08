package slack

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
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

func (c *Client) PostPrivateMessage(userID, env, service string, artifact artifact.Spec, podNotify *http.PodNotifyRequest) error {
	asUser := slack.MsgOptionAsUser(true)
	switch podNotify.State {
	case "CrashLoopBackOff":
		_, _, err := c.client.PostMessage(userID, asUser, crashLoopBackOffErrorMessage(env, service, artifact, podNotify))
		if err != nil {
			return err
		}
	case "CreateContainerConfigError":
		_, _, err := c.client.PostMessage(userID, asUser, createConfigErrorMessage(env, service, artifact, podNotify))
		if err != nil {
			return err
		}
	case "Running", "Ready":
		_, _, err := c.client.PostMessage(userID, asUser, successMessage(env, service, artifact, podNotify))
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown pod state in post private message")
	}
	return nil
}

func successMessage(env, service string, artifact artifact.Spec, podNotify *http.PodNotifyRequest) slack.MsgOption {
	color := "#FF9830"
	if podNotify.State == "Ready" {
		color = "#73bf69"
	}
	podField := slack.AttachmentField{
		Title: "Pod",
		Value: podNotify.Name,
		Short: true,
	}
	statusField := slack.AttachmentField{
		Title: "Status",
		Value: podNotify.State,
		Short: true,
	}
	namespaceField := slack.AttachmentField{
		Title: "Namespace",
		Value: podNotify.Namespace,
		Short: true,
	}
	containersField := slack.AttachmentField{
		Title: "Containers",
		Value: fmt.Sprintf("%d", len(podNotify.Containers)),
		Short: true,
	}
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (artifact: %s)", service, artifact.ID),
		Color:      color,
		MarkdownIn: []string{"text", "fields"},
		Fields:     []slack.AttachmentField{podField, namespaceField, statusField, containersField},
	})
}

func createConfigErrorMessage(env, service string, artifact artifact.Spec, podNotify *http.PodNotifyRequest) slack.MsgOption {
	podField := slack.AttachmentField{
		Title: "Pod",
		Value: podNotify.Name,
		Short: true,
	}
	statusField := slack.AttachmentField{
		Title: "Status",
		Value: podNotify.State,
		Short: true,
	}
	namespaceField := slack.AttachmentField{
		Title: "Namespace",
		Value: podNotify.Namespace,
		Short: true,
	}
	containersField := slack.AttachmentField{
		Title: "Containers",
		Value: fmt.Sprintf("%d", len(podNotify.Containers)),
		Short: true,
	}
	messageField := slack.AttachmentField{
		Title: "Containers",
		Value: podNotify.Message,
		Short: false,
	}
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (artifact: %s)", service, artifact.ID),
		Color:      "#e24d42",
		MarkdownIn: []string{"text", "fields"},
		Fields:     []slack.AttachmentField{podField, namespaceField, statusField, containersField, messageField},
	})
}

func crashLoopBackOffErrorMessage(env, service string, artifact artifact.Spec, podNotify *http.PodNotifyRequest) slack.MsgOption {
	fmt.Printf("%+v", podNotify)
	podField := slack.AttachmentField{
		Title: "Pod",
		Value: podNotify.Name,
		Short: true,
	}
	statusField := slack.AttachmentField{
		Title: "Status",
		Value: podNotify.State,
		Short: true,
	}
	namespaceField := slack.AttachmentField{
		Title: "Namespace",
		Value: podNotify.Namespace,
		Short: true,
	}
	containersField := slack.AttachmentField{
		Title: "Containers",
		Value: fmt.Sprintf("%d", len(podNotify.Containers)),
		Short: true,
	}
	logField := slack.AttachmentField{
		Title: "Logs",
		Value: fmt.Sprintf("```%s```", podNotify.Logs),
		Short: false,
	}
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (artifact: %s)", service, artifact.ID),
		Color:      "#e24d42",
		MarkdownIn: []string{"text", "fields"},
		Fields:     []slack.AttachmentField{podField, namespaceField, statusField, containersField, logField},
	})
}

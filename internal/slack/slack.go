package slack

import (
	"fmt"
	"strings"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

type Client struct {
	client        *slack.Client
	emailMappings map[string]string
}

var (
	// ErrUnknownEmail indicates that an email not from the lunarway.com domain
	// is used and no email mapping exists.
	ErrUnknownEmail = errors.New("not a lunarway email")
)

func NewClient(token string, emailMappings map[string]string) (*Client, error) {
	log.Infof("slack: new client: initialized with emailMappings: %+v", emailMappings)
	slackClient := slack.New(token)
	client := Client{
		client:        slackClient,
		emailMappings: emailMappings,
	}
	return &client, nil
}

func (c *Client) getIdByEmail(email string) (string, error) {
	if !strings.Contains(email, "@lunarway.com") {
		// check for fallback emails
		lwEmail, ok := c.emailMappings[email]
		if !ok {
			log.Errorf("%s is not a Lunar Way email and no mapping exist", email)
			return "", ErrUnknownEmail
		}
		email = lwEmail
	}
	user, err := c.client.GetUserByEmail(email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (c *Client) UpdateSlackBuildStatus(channel, title, titleLink, text, color, timestamp string) (string, string, error) {
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      title,
		TitleLink:  titleLink,
		Color:      color,
		Text:       text,
		MarkdownIn: []string{"text", "fields"},
	})
	respChannel, timestamp, _, err := c.client.UpdateMessage(channel, timestamp, asUser, attachments)
	if err != nil {
		return "", "", err
	}
	return respChannel, timestamp, nil
}

func (c *Client) PostSlackBuildStarted(email, title, titleLink, text, color string) (string, string, error) {
	userID, err := c.getIdByEmail(email)
	if err != nil {
		return "", "", err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      title,
		TitleLink:  titleLink,
		Color:      color,
		Text:       text,
		MarkdownIn: []string{"text", "fields"},
	})

	respChannel, timestamp, err := c.client.PostMessage(userID, asUser, attachments)
	if err != nil {
		return "", "", err
	}
	return respChannel, timestamp, err
}

func (c *Client) PostPrivateMessage(email string, podNotify *http.PodNotifyRequest) error {
	userID, err := c.getIdByEmail(email)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	switch podNotify.State {
	case "CrashLoopBackOff":
		_, _, err := c.client.PostMessage(userID, asUser, crashLoopBackOffErrorMessage(podNotify))
		if err != nil {
			return err
		}
	case "CreateContainerConfigError":
		_, _, err := c.client.PostMessage(userID, asUser, createConfigErrorMessage(podNotify))
		if err != nil {
			return err
		}
	case "Running", "Ready":
		_, _, err := c.client.PostMessage(userID, asUser, successMessage(podNotify))
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown pod state in post private message")
	}
	return nil
}

func successMessage(podNotify *http.PodNotifyRequest) slack.MsgOption {
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":white_check_mark: [%s] %s (%s)", podNotify.Environment, podNotify.Name, podNotify.State),
		Text:       fmt.Sprintf("Artifact id %s", podNotify.ArtifactID),
		Color:      "#73bf69",
		MarkdownIn: []string{"text", "fields"},
	})
}

func createConfigErrorMessage(podNotify *http.PodNotifyRequest) slack.MsgOption {
	messageField := slack.AttachmentField{
		Title: "Error",
		Value: fmt.Sprintf("```%s```", podNotify.Message),
		Short: false,
	}
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":no_entry: [%s] %s (%s)", podNotify.Environment, podNotify.Name, podNotify.State),
		Text:       fmt.Sprintf("Artifact id %s", podNotify.ArtifactID),
		Color:      "#e24d42",
		MarkdownIn: []string{"text", "fields"},
		Fields:     []slack.AttachmentField{messageField},
	})
}

func crashLoopBackOffErrorMessage(podNotify *http.PodNotifyRequest) slack.MsgOption {
	logField := slack.AttachmentField{
		Title: "Logs",
		Value: fmt.Sprintf("```%s```", podNotify.Logs),
		Short: false,
	}
	return slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":no_entry: [%s] %s (%s)", podNotify.Environment, podNotify.Name, podNotify.State),
		Text:       fmt.Sprintf("Artifact id %s", podNotify.ArtifactID),
		Color:      "#e24d42",
		MarkdownIn: []string{"text", "fields"},
		Fields:     []slack.AttachmentField{logField},
	})
}

type ReleaseOptions struct {
	Service       string
	ArtifactID    string
	CommitSHA     string
	CommitLink    string
	CommitMessage string
	CommitAuthor  string
	Releaser      string
	Environment   string
}

func (c *Client) NotifySlackReleasesChannel(options ReleaseOptions) error {
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (%s)", options.Service, options.ArtifactID),
		TitleLink:  options.CommitLink,
		Color:      MsgColorGreen,
		Text:       fmt.Sprintf("*Author:* %s, *Releaser:* %s\n*Message:* _%s_", options.CommitAuthor, options.Releaser, options.CommitMessage),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err := c.client.PostMessage(fmt.Sprintf("#releases-%s", options.Environment), asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

type BuildsOptions struct {
	Service       string
	ArtifactID    string
	Branch        string
	CommitSHA     string
	CommitLink    string
	CommitMessage string
	CommitAuthor  string
	CIJobURL      string
	Color         string
}

func (c *Client) NotifySlackBuildsChannel(options BuildsOptions) error {
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (%s)", options.Service, options.ArtifactID),
		TitleLink:  options.CIJobURL,
		Color:      options.Color,
		Text:       fmt.Sprintf("*Author:* %s (<%s|%s>)\n*Message:* _%s_", options.CommitAuthor, options.CommitLink, options.CommitSHA[0:10], options.CommitMessage),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err := c.client.PostMessage("#builds", asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifySlackPolicyFailed(email, title, errorMessage string) error {
	userID, err := c.getIdByEmail(email)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      title,
		Color:      MsgColorRed,
		Text:       fmt.Sprintf("```%s```", errorMessage),
		MarkdownIn: []string{"text", "fields"},
	})

	_, _, err = c.client.PostMessage(userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) NotifySlackPolicySucceeded(email, title, message string) error {
	userID, err := c.getIdByEmail(email)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      title,
		Color:      MsgColorGreen,
		Text:       message,
		MarkdownIn: []string{"text", "fields"},
	})

	_, _, err = c.client.PostMessage(userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

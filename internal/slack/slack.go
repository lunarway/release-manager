package slack

import (
	"context"
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
	muteOptions   MuteOptions
	emailSuffix   string
}

type MuteOptions struct {
	Flux                bool
	Kubernetes          bool
	Policy              bool
	ReleaseProcessed    bool
	Releases            bool
	BuildStatus         bool
	ReleaseManagerError bool
}

var (
	// ErrUnknownEmail indicates that an email is not from the specified domain
	// and no email mapping exists
	ErrUnknownEmail = errors.New("not an accepted email domain")
)

func NewClient(token string, emailMappings map[string]string, emailSuffix string) (*Client, error) {
	if token == "" {
		log.Infof("slack: skipping: no token, so no slack notification")
		return &Client{
			muteOptions: MuteOptions{
				Flux:                true,
				Kubernetes:          true,
				Policy:              true,
				ReleaseProcessed:    true,
				Releases:            true,
				BuildStatus:         true,
				ReleaseManagerError: true,
			},
		}, nil
	}

	log.Infof("slack: new client: initialized with emailMappings: %+v", emailMappings)
	slackClient := slack.New(token)
	client := Client{
		client:        slackClient,
		emailMappings: emailMappings,
		muteOptions:   MuteOptions{},
	}
	client.emailSuffix = emailSuffix
	return &client, nil
}

func NewMuteableClient(token string, emailMappings map[string]string, muteOptions MuteOptions) (*Client, error) {
	if token == "" {
		log.Infof("slack: skipping: no token, so no slack notification")
		return &Client{
			muteOptions: MuteOptions{
				Flux:                true,
				Kubernetes:          true,
				Policy:              true,
				ReleaseProcessed:    true,
				Releases:            true,
				BuildStatus:         true,
				ReleaseManagerError: true,
			},
		}, nil
	}

	log.Infof("slack: new client: initialized with emailMappings: %+v", emailMappings)
	slackClient := slack.New(token)
	client := Client{
		client:        slackClient,
		emailMappings: emailMappings,
		muteOptions:   muteOptions,
	}
	return &client, nil
}

func (c *Client) getIdByEmail(ctx context.Context, email string) (string, error) {
	if !strings.HasSuffix(email, c.emailSuffix) {
		// check for fallback emails
		companyEmail, ok := c.emailMappings[email]
		if !ok {
			log.WithContext(ctx).Errorf("%s is not a %s email and no mapping exist", email, c.emailSuffix) // todo: what is this and why log + return err
			return "", ErrUnknownEmail
		}
		email = companyEmail
	}
	user, err := c.client.GetUserByEmailContext(ctx, email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (c *Client) UpdateSlackBuildStatus(channel, title, titleLink, text, color, timestamp string) (string, string, error) {
	if c.muteOptions.BuildStatus {
		return "", "", nil
	}

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
	if c.muteOptions.BuildStatus {
		return "", "", nil
	}

	userID, err := c.getIdByEmail(context.Background(), email)
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

type ReleaseOptions struct {
	Service           string
	ArtifactID        string
	CommitSHA         string
	CommitLink        string
	CommitMessage     string
	CommitAuthor      string
	CommitAuthorEmail string
	Releaser          string
	Environment       string
}

func (c *Client) NotifySlackReleasesChannel(ctx context.Context, options ReleaseOptions) error {
	if c.muteOptions.Releases {
		return nil
	}

	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf("%s (%s)", options.Service, options.ArtifactID),
		TitleLink:  options.CommitLink,
		Color:      MsgColorGreen,
		Text:       fmt.Sprintf("*Author:* %s, *Releaser:* %s\n*Message:* _%s_", options.CommitAuthor, options.Releaser, options.CommitMessage),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err := c.client.PostMessageContext(ctx, fmt.Sprintf("#releases-%s", options.Environment), asUser, attachments)
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

func (c *Client) NotifySlackPolicyFailed(ctx context.Context, email, title, errorMessage string) error {
	if c.muteOptions.Policy {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, email)
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

	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) NotifySlackPolicySucceeded(ctx context.Context, email, title, message string) error {
	if c.muteOptions.Policy {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, email)
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

	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) NotifyAuthorEventProcessed(ctx context.Context, options ReleaseOptions) error {
	if c.muteOptions.ReleaseProcessed {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, options.CommitAuthorEmail)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":rocket: Release Manager :white_check_mark:"),
		Color:      MsgColorGreen,
		Text:       fmt.Sprintf("Release for *%s* in %s processed\nArtifact: <%s|*%s*>", options.Service, options.Environment, options.CommitLink, options.ArtifactID),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifyFluxEventProcessed(ctx context.Context, artifactID, env, email, service string) error {
	if c.muteOptions.Flux {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, email)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":flux: Flux (%s) :white_check_mark:", env),
		Color:      MsgColorGreen,
		Text:       fmt.Sprintf("Rollout initiated for *%s*\nArtifact: %s", service, artifactID),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifyFluxErrorEvent(ctx context.Context, artifactID, env, email, service, errorMessage, errorPath string) error {
	if c.muteOptions.Flux {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, email)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":flux: Flux (%s) :no_entry:", env),
		Color:      MsgColorRed,
		Text:       fmt.Sprintf("Error detected for *%s*\nArtifact: %s\nPath: `%s`\n```%s```", service, artifactID, errorPath, errorMessage),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifyK8SDeployEvent(ctx context.Context, event *http.ReleaseEvent) error {
	if c.muteOptions.Kubernetes {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, event.AuthorEmail)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":kubernetes: k8s (%s) :white_check_mark:", event.Environment),
		Color:      MsgColorGreen,
		Text:       fmt.Sprintf("%s deployed\n%d/%d pods are running (%s)\nArtifact: *%s*", event.Name, event.AvailablePods, event.DesiredPods, event.ResourceType, event.ArtifactID),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifyK8SPodErrorEvent(ctx context.Context, event *http.PodErrorEvent) error {
	if c.muteOptions.Kubernetes {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, event.AuthorEmail)
	if err != nil {
		return err
	}
	asUser := slack.MsgOptionAsUser(true)
	var fields []slack.AttachmentField
	for _, container := range event.Errors {
		fields = append(fields, slack.AttachmentField{
			Title: fmt.Sprintf("Container: %s (%s)", container.Name, container.Type),
			Value: fmt.Sprintf("```%s```", container.ErrorMessage),
			Short: false,
		})
	}
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":kubernetes: k8s (%s) :no_entry:", event.Environment),
		Text:       fmt.Sprintf("Pod Error: %s\nArtifact: *%s*", event.PodName, event.ArtifactID),
		Color:      "#e24d42",
		MarkdownIn: []string{"text", "fields"},
		Fields:     fields,
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) NotifyReleaseManagerError(ctx context.Context, msgType, service, environment, branch, namespace, actorEmail string, inputErr error) error {
	if c.muteOptions.ReleaseManagerError {
		return nil
	}
	userID, err := c.getIdByEmail(ctx, actorEmail)
	if err != nil {
		// If user id somehow couldn't be found, post the message to fallback channel
		log.With("actorEmail", actorEmail).Infof("slack: skipping: no user id found, so no slack notification")
		return nil
	}

	asUser := slack.MsgOptionAsUser(true)
	attachments := slack.MsgOptionAttachments(slack.Attachment{
		Title:      fmt.Sprintf(":boom: Release Manager failed :x:"),
		Color:      MsgColorRed,
		Text:       generateSlackMessage(msgType, service, environment, branch, namespace, inputErr),
		MarkdownIn: []string{"text", "fields"},
	})
	_, _, err = c.client.PostMessageContext(ctx, userID, asUser, attachments)
	if err != nil {
		return err
	}
	return nil
}

func generateSlackMessage(msgType, service, environment, branch, namespace string, err error) string {
	switch {
	case msgType == "promote" && service != "" && environment != "" && branch != "":
		return fmt.Sprintf("Failed promoting %s #%s in %s. Validate the options you used and try promoting again.\nError: %s", service, branch, environment, err)
	case msgType == "release.branch" && service != "" && environment != "" && branch != "":
		return fmt.Sprintf("Failed releasing %s #%s to %s. Validate the options you used and try releasing again.\nError: %s", service, branch, environment, err)
	default:
		return fmt.Sprintf("Failed handling event in release manager for:\nEvent: %s\nService: %s\nEnvironment: %s\nBranch: %s\nNamespace: %s\nError: %s", msgType, service, environment, branch, namespace, err)
	}
}

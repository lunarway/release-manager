package slack

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
)

var (
	// ErrFileNotFound indicates that an artifact was not found.
	ErrFileNotFound = errors.New("file not found")
	// ErrNotParsable indicates that an artifact could not be parsed against the
	// artifact specification.
	ErrNotParsable = errors.New("message not parsable")
	// ErrUnknownFields indicates that an artifact contains an unknown field.
	ErrUnknownFields = errors.New("message contains unknown fields")

	MsgColorGreen  = "#73BF69"
	MsgColorYellow = "#FADE2A"
	MsgColorRed    = "#F2495C"
)

type Message struct {
	UserID    string `json:"userId,omitempty"`
	Color     string `json:"color,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Text      string `json:"text,omitempty"`
	Title     string `json:"title,omitempty"`
	TitleLink string `json:"titleLink,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Service   string `json:"service,omitempty"`
}

func Get(path string) (Message, error) {
	m, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Message{}, ErrFileNotFound
		}
		return Message{}, err
	}
	defer m.Close()
	var message Message
	decoder := json.NewDecoder(m)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&message)
	if err != nil {
		_, ok := err.(*json.SyntaxError)
		if ok {
			return Message{}, ErrNotParsable
		}
		// there is no other way to detect this error type unfortunately
		// https://github.com/golang/go/blob/277609f844ed9254d25e975f7cf202d042beecc6/src/encoding/json/decode.go#L739
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			return Message{}, errors.WithMessagef(ErrUnknownFields, "%v", err)
		}
		return Message{}, err
	}
	return message, nil
}

func Persist(path string, message Message) error {
	s, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, "open file")
	}
	defer s.Close()
	err = s.Truncate(0)
	if err != nil {
		return errors.WithMessagef(err, "truncate file '%s'", path)
	}
	_, err = s.Seek(0, 0)
	if err != nil {
		return errors.WithMessagef(err, "reset seek on '%s'", path)
	}
	encode := json.NewEncoder(s)
	encode.SetIndent("", "  ")
	err = encode.Encode(message)
	if err != nil {
		return errors.WithMessage(err, "encode spec to json")
	}
	return nil
}

func Update(path, token string, f func(Message) Message) error {
	m, err := Get(path)
	if err != nil {
		return errors.WithMessagef(err, "read artifact '%s'", path)
	}
	m = f(m)

	// Setup Slack client
	client, err := NewClient(token)
	if err != nil {
		return err
	}

	m.Channel, m.Timestamp, err = client.UpdateSlackBuildStatus(m.Channel, m.Title, m.TitleLink, m.Text, m.Color, m.Timestamp)
	if err != nil {
		return err
	}

	// Persist back to the file
	err = Persist(path, m)
	if err != nil {
		return errors.WithMessagef(err, "persist artifact to '%s'", path)
	}

	return nil
}

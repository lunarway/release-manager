package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

var (
	ErrLoginRequired = errors.New("login required")
	ErrTokenExpired  = errors.New("oauth2: token expired and refresh token is not set")
)

type UserAuthenticator struct {
	conf *oauth2.Config
	autoLogin bool
}

func NewUserAuthenticator(clientID, idpURL string, autoLogin bool) UserAuthenticator {
	conf := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			TokenURL:      fmt.Sprintf("%s/v1/token", idpURL),
			DeviceAuthURL: fmt.Sprintf("%s/v1/device/authorize", idpURL),
		},
		Scopes: []string{"openid profile"},
	}
	return UserAuthenticator{
		conf: conf,
		autoLogin: autoLogin,
	}
}

func (g *UserAuthenticator) Login(ctx context.Context) error {
	response, err := g.conf.DeviceAuth(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("please enter code %s at %s\n", response.UserCode, response.VerificationURIComplete)
	err = browser.OpenURL(response.VerificationURIComplete)
	if err != nil {
		return err
	}

	token, err := g.conf.DeviceAccessToken(ctx, response)
	if err != nil {
		return err
	}
	return storeAccessToken(token)
}

func (g *UserAuthenticator) Access(ctx context.Context) (*http.Client, error) {
	token, err := readAccessToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoginRequired, err)
	}
	if !token.Valid() && g.autoLogin {
		err = g.Login(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrLoginRequired, err)
		}
	}
	return g.conf.Client(ctx, token), nil
}

const tokenFile string = ".Config/release-manager/token.json"

func tokenFilePath() string {
	return filepath.Join(os.Getenv("HOME"), tokenFile)
}

func readAccessToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFilePath())
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	err = json.Unmarshal(data, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func storeAccessToken(token *oauth2.Token) error {
	accessToken, err := json.Marshal(token)
	if err != nil {
		return err
	}
	p := tokenFilePath()
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(accessToken)
	if err != nil {
		return err
	}
	return nil
}

type ClientAuthenticator struct {
	conf *clientcredentials.Config
}

func NewClientAuthenticator(clientID, clientSecret, idpURL, scope string) ClientAuthenticator {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("%s/v1/token", idpURL),
		Scopes:       []string{scope},
	}
	return ClientAuthenticator{
		conf: conf,
	}
}

func (g *ClientAuthenticator) Access(ctx context.Context) (*http.Client, error) {
	return g.conf.Client(ctx), nil
}

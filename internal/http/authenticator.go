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

type TokenStore interface {
	storeAccessToken(token *oauth2.Token) error
	readAccessToken() (*oauth2.Token, error)
}

type DeviceAuthenticator interface {
	DeviceAuth(ctx context.Context, opts ...oauth2.AuthCodeOption) (*oauth2.DeviceAuthResponse, error)
	DeviceAccessToken(ctx context.Context, da *oauth2.DeviceAuthResponse, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	Client(ctx context.Context, t *oauth2.Token) *http.Client
}

type UserAuthenticator struct {
	conf       DeviceAuthenticator
	ts         TokenStore
	autoLogin  bool
	popBrowser bool
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
		conf:       conf,
		ts:         newTokenStore(),
		autoLogin:  autoLogin,
		popBrowser: true,
	}
}

func (g *UserAuthenticator) Login(ctx context.Context) error {
	token, err := g.login(ctx)
	if err != nil {
		return err
	}
	return g.ts.storeAccessToken(token)
}

func (g *UserAuthenticator) login(ctx context.Context) (*oauth2.Token, error) {
	response, err := g.conf.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("please enter code %s at %s\n", response.UserCode, response.VerificationURIComplete)
	if g.popBrowser {
		err = browser.OpenURL(response.VerificationURIComplete)
		if err != nil {
			return nil, err
		}
	}
	return g.conf.DeviceAccessToken(ctx, response)
}

func (g *UserAuthenticator) Access(ctx context.Context) (*http.Client, error) {
	token, err := g.ts.readAccessToken()
	if err != nil {
		if g.autoLogin {
			token, err = g.login(ctx)
			if err != nil {
				return nil, fmt.Errorf("auto login failed: %w: %w", ErrLoginRequired, err)
			}
			err = g.ts.storeAccessToken(token)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("%w: %w", ErrLoginRequired, err)
		}
	}
	if !token.Valid() && g.autoLogin {
		token, err = g.login(ctx)
		if err != nil {
			return nil, fmt.Errorf("auto login failed: %w: %w", ErrLoginRequired, err)
		}
		err = g.ts.storeAccessToken(token)
		if err != nil {
			return nil, err
		}
	}
	return g.conf.Client(ctx, token), nil
}

const tokenFile string = ".Config/release-manager/token.json"

func tokenFilePath() string {
	return filepath.Join(os.Getenv("HOME"), tokenFile)
}

type tokenStore struct {
	tokenFilePath string
}

func newTokenStore() *tokenStore {
	return &tokenStore{
		tokenFilePath: tokenFilePath(),
	}
}

func (ts *tokenStore) readAccessToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(ts.tokenFilePath)
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

func (ts *tokenStore) storeAccessToken(token *oauth2.Token) error {
	accessToken, err := json.Marshal(token)
	if err != nil {
		return err
	}
	dir := filepath.Dir(ts.tokenFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.Create(ts.tokenFilePath)
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

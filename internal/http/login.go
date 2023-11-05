package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Gate struct {
	conf *oauth2.Config
}

func NewGate(clientID, idpURL string) Gate {
	conf := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			TokenURL:      fmt.Sprintf("%s/v1/token", idpURL),
			DeviceAuthURL: fmt.Sprintf("%s/v1/device/authorize", idpURL),
		},
		Scopes: []string{"openid profile"},
	}
	return Gate{
		conf: conf,
	}
}

func (g *Gate) Authenticate() error {
	ctx := context.Background()
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

func (g *Gate) AuthenticatedClient(ctx context.Context) (*http.Client, error) {
	token, err := readAccessToken()
	if err != nil {
		return nil, err
	}
	return g.conf.Client(ctx, token), nil
}

const tokenFile string = ".Config/hamctl/token.json"

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
	if err := os.MkdirAll(dir, 0755); err != nil {
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

type DaemonGate struct {
	conf *clientcredentials.Config
}

func NewDaemonGate(clientID, clientSecret, idpURL string) DaemonGate {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("%s/v1/token", idpURL),
		Scopes:       []string{""},
	}
	return DaemonGate{
		conf: conf,
	}
}

func (g *DaemonGate) AuthenticatedClient(ctx context.Context) (*http.Client, error) {
	return g.conf.Client(ctx), nil
}

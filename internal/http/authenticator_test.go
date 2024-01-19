package http

import (
	"context"
	"net/http"
	"testing"

	"golang.org/x/oauth2"
)

func TestAccess(t *testing.T) {
	ua := UserAuthenticator{
		conf: &HttpAuth{
			DeviceResp: &oauth2.DeviceAuthResponse{
				UserCode:                "YOLO",
				VerificationURIComplete: "localhost",
			},
		},
		ts:         &DummyStore{},
		autoLogin:  true,
		popBrowser: false,
	}

	client, err := ua.Access(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
}

type HttpAuth struct {
	DeviceResp *oauth2.DeviceAuthResponse
}

func (ha *HttpAuth) DeviceAuth(ctx context.Context, opts ...oauth2.AuthCodeOption) (*oauth2.DeviceAuthResponse, error) {
	return ha.DeviceResp, nil
}

func (ha *HttpAuth) DeviceAccessToken(ctx context.Context, da *oauth2.DeviceAuthResponse, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return nil, nil
}

func (ha *HttpAuth) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	return http.DefaultClient
}

type DummyStore struct {
}

func (ds *DummyStore) storeAccessToken(token *oauth2.Token) error {
	return nil
}
func (ds *DummyStore) readAccessToken() (*oauth2.Token, error) {
	return nil, nil
}

package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/oauth2"
)

func TestAccessCallsLoginAndStoresToken(t *testing.T) {
	httpAuth := HttpAuth{
		DeviceResp: &oauth2.DeviceAuthResponse{
			UserCode:                "YOLO",
			VerificationURIComplete: "localhost",
		},
	}
	dummyStore := DummyStore{
		token: nil,
	}
	ua := UserAuthenticator{
		conf:       &httpAuth,
		ts:         &dummyStore,
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
	if dummyStore.tokenRead == false {
		t.Fatal("expected a read of the token")
	}
	if dummyStore.tokenStored == false {
		t.Fatal("expected the token to be stored")
	}
}

type HttpAuth struct {
	DeviceResp *oauth2.DeviceAuthResponse
	token      *oauth2.Token
}

func (ha *HttpAuth) DeviceAuth(ctx context.Context, opts ...oauth2.AuthCodeOption) (*oauth2.DeviceAuthResponse, error) {
	return ha.DeviceResp, nil
}

func (ha *HttpAuth) DeviceAccessToken(ctx context.Context, da *oauth2.DeviceAuthResponse, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return ha.token, nil
}

func (ha *HttpAuth) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	return http.DefaultClient
}

type DummyStore struct {
	token       *oauth2.Token
	tokenStored bool
	tokenRead   bool
}

func (ds *DummyStore) storeAccessToken(token *oauth2.Token) error {
	ds.tokenStored = true
	return nil
}
func (ds *DummyStore) readAccessToken() (*oauth2.Token, error) {
	ds.tokenRead = true
	if ds.token != nil {
		return nil, fmt.Errorf("no token found")
	}
	return ds.token, nil
}

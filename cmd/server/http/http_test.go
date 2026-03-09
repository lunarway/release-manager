package http

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestAuthenticate(t *testing.T) {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	jwksServer, minter := getJwksEndpoint(t)

	issuer := "test-issuer"
	audience := "test-audience"
	serverToken := "server-token"

	tt := []struct {
		name string

		authorization                 string
		status                        int
		expectedRequestContextSubject string
	}{
		{
			name:          "empty authorization",
			authorization: "",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace token",
			authorization: "  ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "non-bearer authorization",
			authorization: "non-bearer-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "empty bearer authorization",
			authorization: "Bearer ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace bearer authorization",
			authorization: "Bearer      ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "wrong bearer authorization",
			authorization: "Bearer another-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "correct hamctl bearer authorization",
			authorization: "Bearer " + serverToken,
			status:        http.StatusOK,
		},
		{
			name:          "invalid bearer token with lots of dots",
			authorization: "Bearer ......................",
			status:        http.StatusUnauthorized,
		},
		{
			name: "valid jwt bearer authorization",
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "sub",
					Issuer:     issuer,
					Audience:   audience,
					IssuedAt:   time.Now().Add(-1 * time.Second),
					Expiration: time.Now().Add(10 * time.Second),
				}),
			),
			status:                        http.StatusOK,
			expectedRequestContextSubject: "sub",
		},
		{
			name: "expired bearer authorization",
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "sub",
					Issuer:     issuer,
					Audience:   audience,
					IssuedAt:   time.Now().Add(-2 * time.Second),
					Expiration: time.Now().Add(-1 * time.Second),
				}),
			),
			status: http.StatusUnauthorized,
		},
		{
			name: "wrong issuer bearer authorization",
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "sub",
					Issuer:     "wrong-issuer",
					Audience:   audience,
					IssuedAt:   time.Now().Add(-1 * time.Second),
					Expiration: time.Now().Add(10 * time.Second),
				}),
			),
			status: http.StatusUnauthorized,
		},
		{
			name: "wrong audience bearer authorization",
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "sub",
					Issuer:     issuer,
					Audience:   "wrong-audience",
					IssuedAt:   time.Now().Add(-1 * time.Second),
					Expiration: time.Now().Add(10 * time.Second),
				}),
			),
			status: http.StatusUnauthorized,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			verifier, err := NewVerifier(context.Background(), jwksServer.URL, issuer, audience)
			require.NoError(t, err, "failed to create verifier")

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			verifier.authentication([]string{serverToken})(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.expectedRequestContextSubject, UserFromContext(req.Context()))
			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}

func TestAuthenticate_withMultipleStaticTokens(t *testing.T) {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	jwksServer, _ := getJwksEndpoint(t)

	issuer := "test-issuer"
	audience := "test-audience"
	firstToken := "first-token"
	secondToken := "second-token"

	tt := []struct {
		name          string
		staticTokens  []string
		authorization string
		status        int
	}{
		{
			name:          "first static token accepted",
			staticTokens:  []string{firstToken, secondToken},
			authorization: "Bearer " + firstToken,
			status:        http.StatusOK,
		},
		{
			name:          "second static token accepted",
			staticTokens:  []string{firstToken, secondToken},
			authorization: "Bearer " + secondToken,
			status:        http.StatusOK,
		},
		{
			name:          "unknown token rejected",
			staticTokens:  []string{firstToken, secondToken},
			authorization: "Bearer unknown-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "empty static tokens array rejects static token",
			staticTokens:  []string{},
			authorization: "Bearer " + firstToken,
			status:        http.StatusUnauthorized,
		},
		{
			name:          "nil static tokens array rejects static token",
			staticTokens:  nil,
			authorization: "Bearer " + firstToken,
			status:        http.StatusUnauthorized,
		},
		{
			name:          "empty string in token array is skipped",
			staticTokens:  []string{"", firstToken, ""},
			authorization: "Bearer " + firstToken,
			status:        http.StatusOK,
		},
		{
			name:          "only empty strings in token array rejects request",
			staticTokens:  []string{"", "", ""},
			authorization: "Bearer " + firstToken,
			status:        http.StatusUnauthorized,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			verifier, err := NewVerifier(context.Background(), jwksServer.URL, issuer, audience)
			require.NoError(t, err, "failed to create verifier")

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			verifier.authentication(tc.staticTokens)(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}

// TestAuthenticate_JWTFallbackWithNoStaticTokens tests that JWT authentication
// works when no static tokens are configured
func TestAuthenticate_JWTFallbackWithNoStaticTokens(t *testing.T) {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	jwksServer, minter := getJwksEndpoint(t)

	issuer := "test-issuer"
	audience := "test-audience"

	tt := []struct {
		name                          string
		staticTokens                  []string
		authorization                 string
		status                        int
		expectedRequestContextSubject string
	}{
		{
			name:         "empty static tokens with valid JWT succeeds",
			staticTokens: []string{},
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "jwt-user",
					Issuer:     issuer,
					Audience:   audience,
					IssuedAt:   time.Now().Add(-1 * time.Second),
					Expiration: time.Now().Add(10 * time.Second),
				}),
			),
			status:                        http.StatusOK,
			expectedRequestContextSubject: "jwt-user",
		},
		{
			name:         "nil static tokens with valid JWT succeeds",
			staticTokens: nil,
			authorization: fmt.Sprintf("Bearer %s",
				minter(t, principal{
					Subject:    "jwt-user",
					Issuer:     issuer,
					Audience:   audience,
					IssuedAt:   time.Now().Add(-1 * time.Second),
					Expiration: time.Now().Add(10 * time.Second),
				}),
			),
			status:                        http.StatusOK,
			expectedRequestContextSubject: "jwt-user",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			verifier, err := NewVerifier(context.Background(), jwksServer.URL, issuer, audience)
			require.NoError(t, err, "failed to create verifier")

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			verifier.authentication(tc.staticTokens)(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.expectedRequestContextSubject, UserFromContext(req.Context()))
			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}

// TestAuthenticate_withInvalidCache tests that jwk cache refresh works as
// intended and that tokens are accepted after a refresh
func TestAuthenticate_withInvalidCache(t *testing.T) {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	testIssuer := "https://auth.dev.lunar.tech/"
	testAudience := "audience"
	testSubject := "subject"

	// Set up the invalid cache
	_, unusedPublic1 := createSigningKey(t, testIssuer)
	unusedJWKKey := jwk.NewSet()
	err := unusedJWKKey.AddKey(unusedPublic1)
	require.NoError(t, err, "add key to unused JWK failed")

	// Set up the valid cache
	validPrivateKey, validPublicKey := createSigningKey(t, testIssuer)
	validJWK := jwk.NewSet()
	err = validJWK.AddKey(validPublicKey)
	require.NoError(t, err, "add key to valid jwk failed")

	// test server will return the valid JWK after first call
	handlerCalled := false
	jwkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !handlerCalled {
			json.NewEncoder(w).Encode(unusedJWKKey)
		} else {
			json.NewEncoder(w).Encode(validJWK)
		}
		// return valid cache second time it's called
		handlerCalled = true
	}))

	signedJwt := createSignedJWTWithKey(t, &principal{
		Subject:    testSubject,
		Issuer:     testIssuer,
		Audience:   testAudience,
		IssuedAt:   time.Now().Add(-1 * time.Minute),
		Expiration: time.Now().Add(time.Minute),
	}, validPrivateKey)

	// Set up the cache
	cache := jwk.NewCache(context.Background())
	err = cache.Register(jwkServer.URL)
	require.NoError(t, err, "failed to register server in cache")

	_, err = cache.Refresh(context.Background(), jwkServer.URL)
	require.NoError(t, err, "failed to refresh cache")

	authenticator, err := NewVerifier(context.Background(), jwkServer.URL, testIssuer, testAudience)
	require.NoError(t, err, "failed to create verified")
	authenticator.jwkCache = cache

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", signedJwt)

	authenticator.authentication([]string{"static-token"})(handler).ServeHTTP(w, req)

	assert.Equal(t, testSubject, UserFromContext(req.Context()))
	assert.Equal(t, http.StatusOK, w.Result().StatusCode, "status code not as expected")
}

func getJwksEndpoint(t *testing.T) (*httptest.Server, func(t *testing.T, principal principal) string) {
	t.Helper()

	jwkSet := jwk.NewSet()

	jwkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jwkSet)
	}))

	privateJwk, publicJwk := createSigningKey(t, jwkServer.URL)
	err := jwkSet.AddKey(publicJwk)
	require.NoError(t, err, "could not add jwk")

	return jwkServer, func(t *testing.T, principal principal) string {
		return getSignedJwt(t, privateJwk, principal)
	}
}

// Given a JWK and a Principal returns a JWT containing the claims in the Principal and signed by the JWK.
func getSignedJwt(t *testing.T, jwk jwk.Key, principal principal) string {
	t.Helper()

	signedJwt := createSignedJWTWithKey(t, &principal, jwk)
	return signedJwt
}

type principal struct {
	Subject    string
	Issuer     string
	Audience   string
	IssuedAt   time.Time
	Expiration time.Time
	Claims     map[string]interface{}
}

func createSignedJWTWithKey(t *testing.T, principal *principal, privateJwkKey jwk.Key) string {
	t.Helper()

	// Create a new JWT
	token := jwt.New()
	token.Set("sub", principal.Subject)
	token.Set("iss", principal.Issuer)
	token.Set("aud", principal.Audience)
	token.Set("iat", principal.IssuedAt.UTC().Unix())
	token.Set("exp", principal.Expiration.UTC().Unix())
	for k, v := range principal.Claims {
		token.Set(k, v)
	}

	// Sign the JWT using the private key
	sig, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, privateJwkKey))
	require.NoError(t, err, "could not sign token")

	return string(sig)
}

func createSigningKey(t *testing.T, issuer string) (jwk.Key, jwk.Key) {
	t.Helper()

	// Create keypair
	privateKey, publicKey, alg := createECDSAKeyPair(t)

	// Create a JWK from the private key
	privateJwkKey, err := jwk.FromRaw(privateKey)
	require.NoError(t, err, "could not create jwk key from private key")

	privateJwkKey.Set("iss", issuer)
	privateJwkKey.Set("alg", jwa.ES256)
	jwk.AssignKeyID(privateJwkKey)

	// Create a JWK from the public key
	publicJwkKey, err := jwk.FromRaw(publicKey)
	require.NoError(t, err, "could not create jwk key from public key")

	publicJwkKey.Set("iss", issuer)
	publicJwkKey.Set("alg", alg)
	jwk.AssignKeyID(publicJwkKey)

	return privateJwkKey, publicJwkKey
}

func createECDSAKeyPair(t *testing.T) (interface{}, interface{}, jwa.SignatureAlgorithm) {
	t.Helper()

	// Generate a new key pair
	keyPair, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "could not generate ecdsa key")

	return keyPair, keyPair.Public(), jwa.ES256
}

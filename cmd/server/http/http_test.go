package http

import (
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

func TestAuthenticate_token(t *testing.T) {
	tt := []struct {
		name          string
		serverToken   string
		authorization string
		status        int
	}{
		{
			name:          "empty authorization",
			serverToken:   "token",
			authorization: "",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace token",
			serverToken:   "token",
			authorization: "  ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "non-bearer authorization",
			serverToken:   "token",
			authorization: "non-bearer-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "empty bearer authorization",
			serverToken:   "token",
			authorization: "Bearer ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace bearer authorization",
			serverToken:   "token",
			authorization: "Bearer      ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "wrong bearer authorization",
			serverToken:   "token",
			authorization: "Bearer another-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "correct bearer authorization",
			serverToken:   "token",
			authorization: "Bearer token",
			status:        http.StatusOK,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			verifier := Verifier{}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			verifier.authentication(tc.serverToken)(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}

func TestAuthenticate_jwt(t *testing.T) {
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
			name: "valid bearer authorization",
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
			verifier, err := NewVerifier(jwksServer.URL, 1*time.Second, issuer, audience)
			require.NoError(t, err, "failed to create verifier")

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			verifier.authentication("")(handler).ServeHTTP(w, req)

			if tc.expectedRequestContextSubject != "" {
				assert.Equal(t, tc.expectedRequestContextSubject, req.Context().Value(AUTH_USER_KEY))
			} else {
				assert.Equal(t, nil, req.Context().Value(AUTH_USER_KEY))
			}
			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
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
	Claims     map[string]string
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

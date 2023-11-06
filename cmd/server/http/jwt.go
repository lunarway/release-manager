package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
)

const keyNotFoundMsg = "failed to find key with key ID"

type JwkCache interface {
	Get(ctx context.Context, url string) (jwk.Set, error)
	Refresh(ctx context.Context, url string) (jwk.Set, error)
}

type Verifier struct {
	jwksLocation string
	issuer       string
	audience     string

	jwkCache JwkCache
}

func NewVerifier(jwksLocation string, jwkFetchTimeout time.Duration, issuer string, audience string) (*Verifier, error) {
	ctx := context.Background()

	cache := jwk.NewCache(ctx)
	err := cache.Register(jwksLocation, jwk.WithMinRefreshInterval(24*time.Hour))
	if err != nil {
		return nil, err
	}
	_, err = cache.Refresh(ctx, jwksLocation)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		jwksLocation: jwksLocation,
		jwkCache:     cache,
		issuer:       issuer,
		audience:     audience,
	}, nil
}

func ParseBearerToken(token string) (string, error) {
	jwt := strings.TrimPrefix(token, "Bearer")

	tokenParts := strings.Split(jwt, ".")

	if len(tokenParts) != 3 {
		return "", errors.New("invalid token format")
	}

	return strings.TrimSpace(jwt), nil
}

const AUTH_USER_KEY = "AUTH_USER_KEY"

// authenticate authenticates the handler against a Bearer token.
//
// If authentication fails a 401 Unauthorized HTTP status is returned with an
// ErrorResponse body.
func (v *Verifier) authentication(token string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bearerToken, err := ParseBearerToken(token)
			if err != nil {
				authorization := r.Header.Get("Authorization")
				t := strings.TrimPrefix(authorization, "Bearer ")
				t = strings.TrimSpace(t)
				if t != token {
					httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
					return
				}
			} else {
				keySet, err := v.jwkCache.Get(context.Background(), v.jwksLocation)
				if err != nil {
					httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
					return
				}

				parsedToken, err := v.verify(bearerToken, keySet)
				if err != nil {
					if strings.Contains(err.Error(), keyNotFoundMsg) {
						freshKeys, err := v.jwkCache.Refresh(context.Background(), v.jwksLocation)
						if err != nil {
							httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
							return
						}
						parsedToken, err = v.verify(bearerToken, freshKeys)
						if err != nil {
							httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
							return
						}
					} else {
						httpinternal.Error(w, "please provide a valid authentication token", http.StatusUnauthorized)
						return
					}
				}
				context.WithValue(r.Context(), AUTH_USER_KEY, parsedToken.Subject())
			}
			h.ServeHTTP(w, r)
		})
	}
}

func (j *Verifier) verify(token string, keySet jwk.Set) (jwt.Token, error) {
	parseOptions := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithVerify(true),
		jwt.WithIssuer(j.issuer),
		jwt.WithAcceptableSkew(time.Second),
	}
	if j.audience != "" {
		parseOptions = append(parseOptions, jwt.WithAudience(j.audience))
	}

	parsedToken, err := jwt.ParseString(token, parseOptions...)
	if err != nil {
		return nil, err
	}

	if parsedToken.Subject() == "" {
		return nil, jwt.ErrMissingRequiredClaim("sub")
	}
	return parsedToken, nil
}

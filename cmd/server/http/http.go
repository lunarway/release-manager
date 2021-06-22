package http

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/google/uuid"
	"github.com/lunarway/release-manager/generated/http/restapi"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
)

type Options struct {
	Port              int
	Timeout           time.Duration
	HamCtlAuthToken   string
	DaemonAuthToken   string
	ArtifactAuthToken string
}

type HandlerFactory func(*operations.ReleaseManagerServerAPIAPI)

func NewServer(opts *Options, tracer tracing.Tracer, handlers []HandlerFactory) error {
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		return fmt.Errorf("load swagger spec: %w", err)
	}
	swaggerAPI := operations.NewReleaseManagerServerAPIAPI(swaggerSpec)
	swaggerAPI.Logger = log.Infof
	swaggerAPI.HamctlAuthorizationTokenAuth = authenticate(opts.HamCtlAuthToken)
	swaggerAPI.DaemonAuthorizationTokenAuth = authenticate(opts.DaemonAuthToken)
	swaggerAPI.ArtifactAuthorizationTokenAuth = authenticate(opts.ArtifactAuthToken)
	// TODO: swaggerAPI.APIAuthorizer = runtime.AuthorizerFunc(func(r *http.Request, i interface{}) error {})

	for _, handler := range handlers {
		handler(swaggerAPI)
	}

	httpServer := http.Server{
		Addr: net.JoinHostPort(hostName(), fmt.Sprintf("%d", opts.Port)),
		Handler: reqrespLogger(
			trace(tracer,
				swaggerAPI.Serve(middleware.PassthroughBuilder),
			),
		),
		ReadTimeout:       opts.Timeout,
		WriteTimeout:      opts.Timeout,
		IdleTimeout:       opts.Timeout,
		ReadHeaderTimeout: opts.Timeout,
	}
	return httpServer.ListenAndServe()
}

func hostName() string {
	if os.Getenv("ENVIRONMENT") == "local" {
		return "localhost"
	}
	return ""
}

// getRequestID returns the request ID of an HTTP request if it is set and
// otherwise generates a new one.
func getRequestID(r *http.Request) string {
	requestID := r.Header.Get("x-request-id")
	if requestID != "" {
		return requestID
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return ""
	}
	r.Header.Set("x-request-id", requestID)
	return id.String()
}

// trace adds an OpenTracing span to the request context.
func trace(tracer tracing.Tracer, h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span, ctx := tracer.FromCtxf(ctx, "http %s %s", r.Method, r.URL.Path)
		defer span.Finish()
		requestID := getRequestID(r)
		ctx = tracing.WithRequestID(ctx, requestID)
		ctx = log.AddContext(ctx, "requestId", requestID)
		*r = *r.WithContext(ctx)
		statusWriter := &statusCodeResponseWriter{w, http.StatusOK}
		h.ServeHTTP(statusWriter, r)
		span.SetTag("request.id", requestID)
		span.SetTag("http.status_code", statusWriter.statusCode)
		span.SetTag("http.url", r.URL.RequestURI())
		span.SetTag("http.method", r.Method)
		if statusWriter.statusCode >= http.StatusInternalServerError {
			span.SetTag("error", true)
		}
		err := ctx.Err()
		if err != nil {
			span.SetTag("error", true)
			span.SetTag("error_message", err.Error())
		}
	})
}

func authenticate(token string) func(token string) (principal interface{}, err error) {
	return func(authorization string) (principal interface{}, err error) {
		t := strings.TrimPrefix(authorization, "Bearer ")
		t = strings.TrimSpace(t)
		if t != token {
			return nil, fmt.Errorf("please provide a valid authentication token")
		}
		return nil, nil
	}
}

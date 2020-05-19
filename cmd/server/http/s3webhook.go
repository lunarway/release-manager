package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
)

func s3webhook() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)

		logger.WithFields("payload", nil).Infof("http: github webhook: payload t")

		w.WriteHeader(http.StatusOK)
	}
}

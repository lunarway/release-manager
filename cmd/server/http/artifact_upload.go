package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/s3storage"
	opentracing "github.com/opentracing/opentracing-go"
)

func artifactUpload(s3storageSvc *s3storage.Service) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)

		logger.WithFields("payload", struct{}{}).Infof("http: github web: ignored")

		w.WriteHeader(http.StatusOK)
	}
}

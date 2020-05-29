package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/artifact"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
)

func createArtifact(payload *payload, artifactWriteStorage ArtifactWriteStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := r.Context()
		logger := log.WithContext(ctx)

		var req httpinternal.ArtifactUploadRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: artifact: create: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		uploadURL, err := artifactWriteStorage.CreateArtifact(req.Artifact)
		if err != nil {
			logger.Errorf("http: artifact: create: storage failed failed creating artifact: %v", err)
			unknownError(w)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = payload.encodeResponse(ctx, w, httpinternal.ArtifactUploadResponse{
			ArtifactUploadURL: uploadURL,
		})
		if err != nil {
			logger.Errorf("http: artifact: create: marshal response failed: %v", err)
		}
	}
}

type ArtifactWriteStorage interface {
	// CreateArtifact creates the artifact in the storage and returns an URL for uploading the artifact zipped
	CreateArtifact(artifactSpec artifact.Spec) (string, error)
}

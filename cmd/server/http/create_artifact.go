package http

import (
	"net/http"

	"github.com/lunarway/release-manager/internal/artifact"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func createArtifact(payload *payload, artifactWriteStorage ArtifactWriteStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.WithContext(ctx)

		var req httpinternal.ArtifactUploadRequest
		err := payload.decodeResponse(ctx, r.Body, &req)
		if err != nil {
			logger.Errorf("http: artifact: create: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}

		logger.Infof("http: artifact: creating artiact for '%s' hash '%x'", req.Artifact.ID, req.MD5)
		uploadURL, err := artifactWriteStorage.CreateArtifact(req.Artifact, req.MD5)
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
	CreateArtifact(artifactSpec artifact.Spec, md5 string) (string, error)
}

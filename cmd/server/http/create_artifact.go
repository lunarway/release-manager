package http

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/release"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
)

func CreateArtifactHandler(artifactWriteStorage ArtifactWriteStorage) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.ReleasePostArtifactCreateHandler = release.PostArtifactCreateHandlerFunc(func(params release.PostArtifactCreateParams, principal interface{}) middleware.Responder {
			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx)

			var (
				service    = *params.Body.Artifact.Service
				artifactID = *params.Body.Artifact.ID
				md5        = *params.Body.Md5
			)

			logger.Infof("http: artifact: creating artifact for '%s/%s' with hash '%x'", service, artifactID, md5)
			uploadURL, err := artifactWriteStorage.CreateArtifact(artifact.Spec{
				ID:      artifactID,
				Service: service,
			}, md5)
			if err != nil {
				logger.Errorf("http: artifact: create: storage failed failed creating artifact: %v", err)
				return release.NewPostArtifactCreateInternalServerError().WithPayload(unknownError())
			}

			return release.NewPostArtifactCreateCreated().WithPayload(&models.CreateArtifactResponse{
				ArtifactUploadURL: uploadURL,
			})
		})
	}
}

type ArtifactWriteStorage interface {
	// CreateArtifact creates the artifact in the storage and returns an URL for uploading the artifact zipped
	CreateArtifact(artifactSpec artifact.Spec, md5 string) (string, error)
}

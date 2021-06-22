package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/release"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/log"
)

func ReleaseHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.ReleasePostReleaseHandler = release.PostReleaseHandlerFunc(func(params release.PostReleaseParams, principal interface{}) middleware.Responder {
			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx)

			var (
				service        = *params.Body.Service
				committerName  = *params.Body.CommitterName
				committerEmail = *params.Body.CommitterEmail
				environment    = *params.Body.Environment
				artifactID     = *params.Body.ArtifactID
				intent         = intent.Intent{
					Type: *params.Body.Intent.Type,
					ReleaseBranch: intent.ReleaseBranchIntent{
						Branch: params.Body.Intent.ReleaseBranch.Branch,
					},
					Promote: intent.PromoteIntent{
						FromEnvironment: params.Body.Intent.Promote.FromEnvironment,
					},
					Rollback: intent.RollbackIntent{
						PreviousArtifactID: params.Body.Intent.Rollback.PreviousArtifactID,
					},
				}
			)

			logger = logger.WithFields(
				"service", service,
				"req", params.Body,
				"intent", intent,
			)

			logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': releasing artifact", service, environment, artifactID)
			releaseID, err := flowSvc.ReleaseArtifactID(ctx, flow.Actor{
				Name:  committerName,
				Email: committerEmail,
			}, environment, service, artifactID, intent)

			var statusString string
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release cancelled", service, environment, artifactID)
					return release.NewPostReleaseBadRequest().
						WithPayload(cancelled())
				}
				switch errorCause(err) {
				case flow.ErrReleaseProhibited:
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release rejected: branch prohibited in environment: %v", service, environment, artifactID, err)
					return release.NewPostReleaseBadRequest().
						WithPayload(badRequest("cannot release %s to environment '%s' due to branch restriction policy", intent.AsArtifactWithIntent(artifactID), environment))
				case flow.ErrNothingToRelease:
					statusString = fmt.Sprintf("Environment '%s' is already up-to-date", environment)
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release skipped: environment up to date: %v", service, environment, artifactID, err)
				case flow.ErrArtifactNotFound:
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release rejected: %v", service, environment, artifactID, err)
					return release.NewPostReleaseBadRequest().
						WithPayload(badRequest("%s not found for service '%s'", intent.AsArtifactWithIntent(artifactID), service))
				case git.ErrBranchBehindOrigin:
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': %v", service, environment, artifactID, err)
					return release.NewPostReleaseServiceUnavailable().
						WithPayload(&models.ErrorResponse{
							Message: fmt.Sprintf("could not release %s right now. Please try again in a moment.", intent.AsArtifactWithIntent(artifactID)),
							Status:  http.StatusServiceUnavailable,
						})
				case artifact.ErrFileNotFound:
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release rejected: %v", service, environment, artifactID, err)
					return release.NewPostReleaseBadRequest().
						WithPayload(badRequest("%s not found for service '%s'", intent.AsArtifactWithIntent(artifactID), service))
				case flow.ErrUnknownEnvironment:
					logger.Infof("http: release: service '%s' environment '%s': release rejected: %v", service, environment, err)
					return release.NewPostReleaseBadRequest().
						WithPayload(badRequest("unknown environment: %s", environment))
				case flow.ErrUnknownConfiguration:
					logger.Infof("http: release: service '%s' environment '%s' artifact id '%s': release rejected: source configuration not found: %v", service, environment, artifactID, err)
					return release.NewPostReleaseBadRequest().
						WithPayload(badRequest("configuration for environment '%s' not found for service '%s'. Is the environment specified in 'shuttle.yaml'?", environment, service))
				default:
					logger.Errorf("http: release: service '%s' environment '%s' artifact id '%s': release failed: %v", service, environment, artifactID, err)
					return release.NewPostReleaseInternalServerError().
						WithPayload(unknownError())
				}
			}

			return release.NewPostReleaseOK().
				WithPayload(&models.ReleaseResponse{
					Service:       service,
					ReleaseID:     releaseID,
					ToEnvironment: environment,
					Tag:           releaseID,
					Status:        statusString,
				})
		})
	}
}

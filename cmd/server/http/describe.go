package http

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/status"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
)

func DescribeReleaseHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.StatusGetDescribeReleaseServiceEnvironmentHandler = status.GetDescribeReleaseServiceEnvironmentHandlerFunc(func(params status.GetDescribeReleaseServiceEnvironmentParams, principal interface{}) middleware.Responder {
			var (
				service     = params.Service
				environment = params.Environment
				namespace   = params.Namespace
				count       = *params.Count
			)

			ctx := params.HTTPRequest.Context()

			logger := log.WithContext(ctx).WithFields("service", service, "environment", environment, "namespace", namespace)
			resp, err := flowSvc.DescribeRelease(ctx, environment, service, int(count))
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: describe release: service '%s' environment '%s': request cancelled", service, environment)
					return status.NewGetDescribeReleaseServiceEnvironmentBadRequest().
						WithPayload(cancelled())
				}
				switch errorCause(err) {
				case artifact.ErrFileNotFound:
					return status.NewGetDescribeReleaseServiceEnvironmentBadRequest().
						WithPayload(badRequest("no release of service '%s' available in environment '%s'. Are you missing a namespace?", service, environment))
				default:
					logger.Errorf("http: describe release: service '%s' environment '%s': failed: %v", service, environment, err)
					return status.NewGetDescribeReleaseServiceEnvironmentInternalServerError().
						WithPayload(unknownError())
				}
			}

			var releases []*models.DescribeReleaseResponseReleasesItems0
			for _, release := range resp.Releases {
				releases = append(releases, &models.DescribeReleaseResponseReleasesItems0{
					ReleaseIndex:    int64(release.ReleaseIndex),
					Artifact:        mapArtifactToHTTP(release.Artifact),
					ReleasedAt:      strfmt.Date(release.ReleasedAt),
					ReleasedByName:  release.ReleasedByName,
					ReleasedByEmail: release.ReleasedByEmail,
					Intent:          mapIntent(release.Intent),
				})
			}

			return status.NewGetDescribeReleaseServiceEnvironmentOK().WithPayload(&models.DescribeReleaseResponse{
				Service:     service,
				Environment: environment,
				Releases:    releases,
			})
		})
	}
}

func DescribeArtifactHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.StatusGetDescribeArtifactServiceHandler = status.GetDescribeArtifactServiceHandlerFunc(func(params status.GetDescribeArtifactServiceParams, principal interface{}) middleware.Responder {
			var (
				service = params.Service
				count   = *params.Count
				branch  = ""
			)
			if params.Branch != nil {
				branch = *params.Branch
			}

			ctx := params.HTTPRequest.Context()

			logger := log.WithContext(ctx).WithFields("service", service, "count", count, "branch", branch)

			resp, err := flowSvc.DescribeArtifact(ctx, service, int(count), branch)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: describe artifact: service '%s': request cancelled", service)
					return status.NewGetDescribeArtifactServiceBadRequest().WithPayload(cancelled())
				}
				switch errorCause(err) {
				case flow.ErrArtifactNotFound:
					return status.NewGetDescribeArtifactServiceBadRequest().WithPayload(badRequest("no artifacts available for service '%s'.", service))
				default:
					logger.Errorf("http: describe artifact: service '%s': failed: %v", service, err)
					return status.NewGetDescribeArtifactServiceInternalServerError().WithPayload(unknownError())
				}
			}

			var httpArtifacts []*models.Artifact
			for _, artifact := range resp {
				httpArtifacts = append(httpArtifacts, mapArtifactToHTTP(artifact))
			}

			return status.NewGetDescribeArtifactServiceOK().WithPayload(&models.DescribeArtifactResponse{
				Service:   service,
				Artifacts: httpArtifacts,
			})
		})
	}
}

func DescribeLatestArtifactsHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.StatusGetDescribeLatestArtifactServiceHandler = status.GetDescribeLatestArtifactServiceHandlerFunc(func(params status.GetDescribeLatestArtifactServiceParams, principal interface{}) middleware.Responder {
			var (
				service = params.Service
				branch  = params.Branch
			)

			ctx := params.HTTPRequest.Context()

			logger := log.WithContext(ctx).WithFields("service", service, "branch", branch)

			resp, err := flowSvc.DescribeLatestArtifact(ctx, service, branch)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: describe latest artifact: service '%s': request cancelled", service)
					return status.NewGetDescribeLatestArtifactServiceBadRequest().WithPayload(cancelled())
				}
				switch errorCause(err) {
				case flow.ErrArtifactNotFound:
					return status.NewGetDescribeLatestArtifactServiceBadRequest().WithPayload(badRequest("no artifacts available for service '%s' and branch '%s'.", service, branch))
				default:
					logger.Errorf("http: describe latest artifact: service '%s' and branch '%s': failed: %v", service, branch, err)
					return status.NewGetDescribeLatestArtifactServiceInternalServerError().WithPayload(unknownError())
				}
			}

			return status.NewGetDescribeLatestArtifactServiceOK().WithPayload(&models.DescribeArtifactResponse{
				Service:   service,
				Artifacts: []*models.Artifact{mapArtifactToHTTP(resp)},
			})
		})
	}
}

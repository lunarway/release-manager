package http

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/status"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
)

func StatusHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.StatusGetStatusHandler = status.GetStatusHandlerFunc(func(params status.GetStatusParams, principal interface{}) middleware.Responder {

			service := params.Service
			var namespace string
			if params.Namespace != nil {
				namespace = *params.Namespace
			}

			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx).WithFields("service", service, "namespace", namespace)
			s, err := flowSvc.Status(ctx, namespace, service)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: status: get status cancelled: service '%s'", service)
					return status.NewGetStatusBadRequest().WithPayload(cancelled())
				}
				logger.Errorf("http: status: get status failed: service '%s': %v", service, err)
				return status.NewGetStatusInternalServerError().WithPayload(unknownError())
			}

			dev := &models.EnvironmentStatus{
				Message:               s.Dev.Message,
				Author:                s.Dev.Author,
				Tag:                   s.Dev.Tag,
				Committer:             s.Dev.Committer,
				Date:                  convertTimeToEpoch(s.Dev.Date),
				BuildURL:              s.Dev.BuildURL,
				HighVulnerabilities:   s.Dev.HighVulnerabilities,
				MediumVulnerabilities: s.Dev.MediumVulnerabilities,
				LowVulnerabilities:    s.Dev.LowVulnerabilities,
			}

			staging := &models.EnvironmentStatus{
				Message:               s.Staging.Message,
				Author:                s.Staging.Author,
				Tag:                   s.Staging.Tag,
				Committer:             s.Staging.Committer,
				Date:                  convertTimeToEpoch(s.Staging.Date),
				BuildURL:              s.Staging.BuildURL,
				HighVulnerabilities:   s.Staging.HighVulnerabilities,
				MediumVulnerabilities: s.Staging.MediumVulnerabilities,
				LowVulnerabilities:    s.Staging.LowVulnerabilities,
			}

			prod := &models.EnvironmentStatus{
				Message:               s.Prod.Message,
				Author:                s.Prod.Author,
				Tag:                   s.Prod.Tag,
				Committer:             s.Prod.Committer,
				Date:                  convertTimeToEpoch(s.Prod.Date),
				BuildURL:              s.Prod.BuildURL,
				HighVulnerabilities:   s.Prod.HighVulnerabilities,
				MediumVulnerabilities: s.Prod.MediumVulnerabilities,
				LowVulnerabilities:    s.Prod.LowVulnerabilities,
			}

			return status.NewGetStatusOK().WithPayload(&models.StatusResponse{
				DefaultNamespaces: s.DefaultNamespaces,
				Dev:               dev,
				Staging:           staging,
				Prod:              prod,
			})
		})
	}
}

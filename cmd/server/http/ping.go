package http

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/internal_swagger"
)

func PingHandler() HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.InternalSwaggerGetPingHandler = internal_swagger.GetPingHandlerFunc(func(gpp internal_swagger.GetPingParams) middleware.Responder {
			return internal_swagger.NewGetPingOK().WithPayload("pong")
		})
	}
}

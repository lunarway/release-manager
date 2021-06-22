// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/internal_swagger"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/policies"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/release"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/status"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/webhook"
)

//go:generate swagger generate server --target ../../http --name ReleaseManagerServerAPI --spec ../../../api/swagger.yaml --principal interface{} --exclude-main

func configureFlags(api *operations.ReleaseManagerServerAPIAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.ReleaseManagerServerAPIAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()
	api.TxtProducer = runtime.TextProducer()

	// Applies when the "Authorization" header is set
	if api.ArtifactAuthorizationTokenAuth == nil {
		api.ArtifactAuthorizationTokenAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (artifactAuthorizationToken) Authorization from header param [Authorization] has not yet been implemented")
		}
	}
	// Applies when the "Authorization" header is set
	if api.DaemonAuthorizationTokenAuth == nil {
		api.DaemonAuthorizationTokenAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (daemonAuthorizationToken) Authorization from header param [Authorization] has not yet been implemented")
		}
	}
	// Applies when the "Authorization" header is set
	if api.HamctlAuthorizationTokenAuth == nil {
		api.HamctlAuthorizationTokenAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (hamctlAuthorizationToken) Authorization from header param [Authorization] has not yet been implemented")
		}
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()

	if api.PoliciesDeletePoliciesHandler == nil {
		api.PoliciesDeletePoliciesHandler = policies.DeletePoliciesHandlerFunc(func(params policies.DeletePoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation policies.DeletePolicies has not yet been implemented")
		})
	}
	if api.StatusGetDescribeArtifactServiceHandler == nil {
		api.StatusGetDescribeArtifactServiceHandler = status.GetDescribeArtifactServiceHandlerFunc(func(params status.GetDescribeArtifactServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation status.GetDescribeArtifactService has not yet been implemented")
		})
	}
	if api.StatusGetDescribeLatestArtifactServiceHandler == nil {
		api.StatusGetDescribeLatestArtifactServiceHandler = status.GetDescribeLatestArtifactServiceHandlerFunc(func(params status.GetDescribeLatestArtifactServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation status.GetDescribeLatestArtifactService has not yet been implemented")
		})
	}
	if api.StatusGetDescribeReleaseServiceEnvironmentHandler == nil {
		api.StatusGetDescribeReleaseServiceEnvironmentHandler = status.GetDescribeReleaseServiceEnvironmentHandlerFunc(func(params status.GetDescribeReleaseServiceEnvironmentParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation status.GetDescribeReleaseServiceEnvironment has not yet been implemented")
		})
	}
	if api.InternalSwaggerGetPingHandler == nil {
		api.InternalSwaggerGetPingHandler = internal_swagger.GetPingHandlerFunc(func(params internal_swagger.GetPingParams) middleware.Responder {
			return middleware.NotImplemented("operation internal_swagger.GetPing has not yet been implemented")
		})
	}
	if api.PoliciesGetPoliciesHandler == nil {
		api.PoliciesGetPoliciesHandler = policies.GetPoliciesHandlerFunc(func(params policies.GetPoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation policies.GetPolicies has not yet been implemented")
		})
	}
	if api.StatusGetStatusHandler == nil {
		api.StatusGetStatusHandler = status.GetStatusHandlerFunc(func(params status.GetStatusParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation status.GetStatus has not yet been implemented")
		})
	}
	if api.PoliciesPatchPolicyAutoReleaseHandler == nil {
		api.PoliciesPatchPolicyAutoReleaseHandler = policies.PatchPolicyAutoReleaseHandlerFunc(func(params policies.PatchPolicyAutoReleaseParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation policies.PatchPolicyAutoRelease has not yet been implemented")
		})
	}
	if api.PoliciesPatchPolicyBranchRestrictionHandler == nil {
		api.PoliciesPatchPolicyBranchRestrictionHandler = policies.PatchPolicyBranchRestrictionHandlerFunc(func(params policies.PatchPolicyBranchRestrictionParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation policies.PatchPolicyBranchRestriction has not yet been implemented")
		})
	}
	if api.ReleasePostArtifactCreateHandler == nil {
		api.ReleasePostArtifactCreateHandler = release.PostArtifactCreateHandlerFunc(func(params release.PostArtifactCreateParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation release.PostArtifactCreate has not yet been implemented")
		})
	}
	if api.ReleasePostReleaseHandler == nil {
		api.ReleasePostReleaseHandler = release.PostReleaseHandlerFunc(func(params release.PostReleaseParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation release.PostRelease has not yet been implemented")
		})
	}
	if api.WebhookPostWebhookDaemonK8sDeployHandler == nil {
		api.WebhookPostWebhookDaemonK8sDeployHandler = webhook.PostWebhookDaemonK8sDeployHandlerFunc(func(params webhook.PostWebhookDaemonK8sDeployParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation webhook.PostWebhookDaemonK8sDeploy has not yet been implemented")
		})
	}
	if api.WebhookPostWebhookDaemonK8sErrorHandler == nil {
		api.WebhookPostWebhookDaemonK8sErrorHandler = webhook.PostWebhookDaemonK8sErrorHandlerFunc(func(params webhook.PostWebhookDaemonK8sErrorParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation webhook.PostWebhookDaemonK8sError has not yet been implemented")
		})
	}
	if api.WebhookPostWebhookDaemonK8sJoberrorHandler == nil {
		api.WebhookPostWebhookDaemonK8sJoberrorHandler = webhook.PostWebhookDaemonK8sJoberrorHandlerFunc(func(params webhook.PostWebhookDaemonK8sJoberrorParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation webhook.PostWebhookDaemonK8sJoberror has not yet been implemented")
		})
	}
	if api.WebhookPostWebhookGithubHandler == nil {
		api.WebhookPostWebhookGithubHandler = webhook.PostWebhookGithubHandlerFunc(func(params webhook.PostWebhookGithubParams) middleware.Responder {
			return middleware.NotImplemented("operation webhook.PostWebhookGithub has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}

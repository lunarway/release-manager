package http

import (
	"context"
	"errors"
	"regexp/syntax"

	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/policies"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/policy"
)

func ApplyAutoReleasePolicyHandler(policySvc *policy.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.PoliciesPatchPolicyAutoReleaseHandler = policies.PatchPolicyAutoReleaseHandlerFunc(func(params policies.PatchPolicyAutoReleaseParams, principal interface{}) middleware.Responder {

			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx)

			var (
				service        = *params.Body.Service
				branch         = *params.Body.Branch
				environment    = *params.Body.Environment
				committerName  = *params.Body.CommitterName
				committerEmail = *params.Body.CommitterEmail
			)

			logger = logger.WithFields("service", service, "req", params.Body)
			logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release policy started", service, branch, environment)
			id, err := policySvc.ApplyAutoRelease(ctx, policy.Actor{
				Name:  committerName,
				Email: committerEmail,
			}, service, branch, environment)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release cancelled", service, branch, environment)
					return policies.NewPatchPolicyAutoReleaseBadRequest().
						WithPayload(cancelled())
				}
				switch errorCause(err) {
				case policy.ErrConflict:
					logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release rejected: conflicts with another policy: %v", service, branch, environment, err)
					return policies.NewPatchPolicyAutoReleaseBadRequest().
						WithPayload(badRequest("policy conflicts with another policy"))
				case git.ErrBranchBehindOrigin:
					logger.Infof("http: policy: apply: service '%s' branch '%s' environment '%s': %v", service, branch, environment, err)
					return policies.NewPatchPolicyAutoReleaseServiceUnavailable().
						WithPayload(unavailable("could not apply policy right now. Please try again in a moment."))
				default:
					logger.Errorf("http: policy: apply: service '%s' branch '%s' environment '%s': apply auto-release failed: %v", service, branch, environment, err)
					return policies.NewPatchPolicyAutoReleaseInternalServerError().
						WithPayload(unknownError())
				}
			}

			return policies.NewPatchPolicyAutoReleaseCreated().
				WithPayload(&models.ApplyAutoReleasePolicyResponse{
					ID:          id,
					Service:     service,
					Branch:      branch,
					Environment: environment,
				})
		})
	}
}

func ApplyBranchRestrictionPolicyHandler(policySvc *policy.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.PoliciesPatchPolicyBranchRestrictionHandler = policies.PatchPolicyBranchRestrictionHandlerFunc(func(params policies.PatchPolicyBranchRestrictionParams, principal interface{}) middleware.Responder {
			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx)

			var (
				service        = *params.Body.Service
				branchRegex    = *params.Body.BranchRegex
				environment    = *params.Body.Environment
				committerName  = *params.Body.CommitterName
				committerEmail = *params.Body.CommitterEmail
			)

			logger = logger.WithFields("service", service, "req", params.Body)
			logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction policy started", service, branchRegex, environment)
			id, err := policySvc.ApplyBranchRestriction(ctx, policy.Actor{
				Name:  committerName,
				Email: committerEmail,
			}, service, branchRegex, environment)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction cancelled", service, branchRegex, environment)
					return policies.NewPatchPolicyBranchRestrictionBadRequest().
						WithPayload(cancelled())
				}
				var regexErr *syntax.Error
				if errors.As(err, &regexErr) {
					logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction: invalid branch regex: %v", service, branchRegex, environment, err)
					return policies.NewPatchPolicyBranchRestrictionBadRequest().
						WithPayload(badRequest("branch regex not valid: %v", regexErr))
				}
				switch errorCause(err) {
				case policy.ErrConflict:
					logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction rejected: conflicts with another policy: %v", service, branchRegex, environment, err)
					return policies.NewPatchPolicyBranchRestrictionBadRequest().
						WithPayload(badRequest("policy conflicts with another policy"))
				case git.ErrBranchBehindOrigin:
					logger.Infof("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction: %v", service, branchRegex, environment, err)
					return policies.NewPatchPolicyBranchRestrictionServiceUnavailable().
						WithPayload(unavailable("could not apply policy right now. Please try again in a moment."))
				default:
					logger.Errorf("http: policy: apply: service '%s' branch regex '%s' environment '%s': apply branch-restriction failed: %v", service, branchRegex, environment, err)
					return policies.NewPatchPolicyBranchRestrictionInternalServerError().WithPayload(unknownError())
				}
			}

			return policies.NewPatchPolicyBranchRestrictionCreated().WithPayload(&models.ApplyBranchRestrictionPolicyResponce{
				ID:          id,
				Service:     service,
				BranchRegex: branchRegex,
				Environment: environment,
			})
		})
	}
}

func ListPoliciesHandler(policySvc *policy.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.PoliciesGetPoliciesHandler = policies.GetPoliciesHandlerFunc(func(params policies.GetPoliciesParams, principal interface{}) middleware.Responder {

			service := params.Service

			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx).WithFields("service", service)

			foundPolicies, err := policySvc.Get(ctx, service)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Infof("http: policy: list: service '%s': get policies cancelled", service)
					return policies.NewGetPoliciesBadRequest().WithPayload(cancelled())
				}
				if errorCause(err) == policy.ErrNotFound {
					return policies.NewGetPoliciesNotFound().WithPayload(notFound("no policiex exist"))
				}
				logger.Errorf("http: policy: list: service '%s': get policies failed: %v", service, err)
				return policies.NewGetPoliciesInternalServerError().WithPayload(unknownError())
			}

			return policies.NewGetPoliciesOK().WithPayload(&models.GetPoliciesResponse{
				Service:            foundPolicies.Service,
				AutoReleases:       mapAutoReleasePolicies(foundPolicies.AutoReleases),
				BranchRestrictions: mapBranchRestrictionPolicies(foundPolicies.BranchRestrictions),
			})
		})
	}
}

func DeletePoliciesHandler(policySvc *policy.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.PoliciesDeletePoliciesHandler = policies.DeletePoliciesHandlerFunc(func(params policies.DeletePoliciesParams, principal interface{}) middleware.Responder {
			ctx := params.HTTPRequest.Context()
			logger := log.WithContext(ctx)

			var (
				ids            = params.Body.PolicyIds
				service        = *params.Body.Service
				committerName  = *params.Body.CommitterName
				committerEmail = *params.Body.CommitterEmail
			)

			logger = logger.WithFields("service", service, "req", params.Body)

			deleted, err := policySvc.Delete(ctx, policy.Actor{
				Name:  committerName,
				Email: committerEmail,
			}, service, ids)
			if err != nil {
				if ctx.Err() == context.Canceled {
					logger.Errorf("http: policy: delete: service '%s' ids %v: delete cancelled", service, ids)
					return policies.NewDeletePoliciesBadRequest().WithPayload(cancelled())
				}
				switch errorCause(err) {
				case policy.ErrNotFound:
					return policies.NewDeletePoliciesNotFound().WithPayload(notFound("no policies exist"))
				case git.ErrBranchBehindOrigin:
					logger.Infof("http: policy: delete: service '%s' ids %v: %v", service, ids, err)
					return policies.NewDeletePoliciesServiceUnavailable().WithPayload(unavailable("could not delete policy right now. Please try again in a moment."))
				default:
					logger.Errorf("http: policy: delete: service '%s' ids %v: delete failed: %v", service, ids, err)
					return policies.NewDeletePoliciesInternalServerError().WithPayload(unknownError())
				}
			}

			return policies.NewDeletePoliciesOK().WithPayload(&models.DeletePoliciesResponse{
				Service: service,
				Count:   int64(deleted),
			})
		})
	}
}

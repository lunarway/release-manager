// Code generated by go-swagger; DO NOT EDIT.

package policies

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PatchPolicyAutoReleaseHandlerFunc turns a function with the right signature into a patch policy auto release handler
type PatchPolicyAutoReleaseHandlerFunc func(PatchPolicyAutoReleaseParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn PatchPolicyAutoReleaseHandlerFunc) Handle(params PatchPolicyAutoReleaseParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// PatchPolicyAutoReleaseHandler interface for that can handle valid patch policy auto release params
type PatchPolicyAutoReleaseHandler interface {
	Handle(PatchPolicyAutoReleaseParams, interface{}) middleware.Responder
}

// NewPatchPolicyAutoRelease creates a new http.Handler for the patch policy auto release operation
func NewPatchPolicyAutoRelease(ctx *middleware.Context, handler PatchPolicyAutoReleaseHandler) *PatchPolicyAutoRelease {
	return &PatchPolicyAutoRelease{Context: ctx, Handler: handler}
}

/* PatchPolicyAutoRelease swagger:route PATCH /policy/auto-release policies patchPolicyAutoRelease

Apply an auto-release policy

*/
type PatchPolicyAutoRelease struct {
	Context *middleware.Context
	Handler PatchPolicyAutoReleaseHandler
}

func (o *PatchPolicyAutoRelease) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPatchPolicyAutoReleaseParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		*r = *aCtx
	}
	var principal interface{}
	if uprinc != nil {
		principal = uprinc.(interface{}) // this is really a interface{}, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}

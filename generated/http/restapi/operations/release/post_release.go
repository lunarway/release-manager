// Code generated by go-swagger; DO NOT EDIT.

package release

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PostReleaseHandlerFunc turns a function with the right signature into a post release handler
type PostReleaseHandlerFunc func(PostReleaseParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn PostReleaseHandlerFunc) Handle(params PostReleaseParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// PostReleaseHandler interface for that can handle valid post release params
type PostReleaseHandler interface {
	Handle(PostReleaseParams, interface{}) middleware.Responder
}

// NewPostRelease creates a new http.Handler for the post release operation
func NewPostRelease(ctx *middleware.Context, handler PostReleaseHandler) *PostRelease {
	return &PostRelease{Context: ctx, Handler: handler}
}

/* PostRelease swagger:route POST /release release postRelease

Release an artifact

*/
type PostRelease struct {
	Context *middleware.Context
	Handler PostReleaseHandler
}

func (o *PostRelease) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPostReleaseParams()
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
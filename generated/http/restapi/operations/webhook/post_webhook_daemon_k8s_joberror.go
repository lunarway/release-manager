// Code generated by go-swagger; DO NOT EDIT.

package webhook

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PostWebhookDaemonK8sJoberrorHandlerFunc turns a function with the right signature into a post webhook daemon k8s joberror handler
type PostWebhookDaemonK8sJoberrorHandlerFunc func(PostWebhookDaemonK8sJoberrorParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn PostWebhookDaemonK8sJoberrorHandlerFunc) Handle(params PostWebhookDaemonK8sJoberrorParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// PostWebhookDaemonK8sJoberrorHandler interface for that can handle valid post webhook daemon k8s joberror params
type PostWebhookDaemonK8sJoberrorHandler interface {
	Handle(PostWebhookDaemonK8sJoberrorParams, interface{}) middleware.Responder
}

// NewPostWebhookDaemonK8sJoberror creates a new http.Handler for the post webhook daemon k8s joberror operation
func NewPostWebhookDaemonK8sJoberror(ctx *middleware.Context, handler PostWebhookDaemonK8sJoberrorHandler) *PostWebhookDaemonK8sJoberror {
	return &PostWebhookDaemonK8sJoberror{Context: ctx, Handler: handler}
}

/* PostWebhookDaemonK8sJoberror swagger:route POST /webhook/daemon/k8s/joberror webhook postWebhookDaemonK8sJoberror

Daemon webhook for kubernetes job error events

*/
type PostWebhookDaemonK8sJoberror struct {
	Context *middleware.Context
	Handler PostWebhookDaemonK8sJoberrorHandler
}

func (o *PostWebhookDaemonK8sJoberror) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPostWebhookDaemonK8sJoberrorParams()
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

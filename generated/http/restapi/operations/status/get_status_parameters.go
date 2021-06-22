// Code generated by go-swagger; DO NOT EDIT.

package status

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// NewGetStatusParams creates a new GetStatusParams object
//
// There are no default values defined in the spec.
func NewGetStatusParams() GetStatusParams {

	return GetStatusParams{}
}

// GetStatusParams contains all the bound params for the get status operation
// typically these are obtained from a http.Request
//
// swagger:parameters GetStatus
type GetStatusParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Namespace to find release in
	  In: query
	*/
	Namespace *string
	/*Service to find releases for
	  Required: true
	  In: query
	*/
	Service string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetStatusParams() beforehand.
func (o *GetStatusParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qNamespace, qhkNamespace, _ := qs.GetOK("namespace")
	if err := o.bindNamespace(qNamespace, qhkNamespace, route.Formats); err != nil {
		res = append(res, err)
	}

	qService, qhkService, _ := qs.GetOK("service")
	if err := o.bindService(qService, qhkService, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindNamespace binds and validates parameter Namespace from query.
func (o *GetStatusParams) bindNamespace(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		return nil
	}
	o.Namespace = &raw

	return nil
}

// bindService binds and validates parameter Service from query.
func (o *GetStatusParams) bindService(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("service", "query", rawData)
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false

	if err := validate.RequiredString("service", "query", raw); err != nil {
		return err
	}
	o.Service = raw

	return nil
}
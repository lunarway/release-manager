// Code generated by go-swagger; DO NOT EDIT.

package policies

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

// NewGetPoliciesParams creates a new GetPoliciesParams object
//
// There are no default values defined in the spec.
func NewGetPoliciesParams() GetPoliciesParams {

	return GetPoliciesParams{}
}

// GetPoliciesParams contains all the bound params for the get policies operation
// typically these are obtained from a http.Request
//
// swagger:parameters GetPolicies
type GetPoliciesParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Service to find policies for
	  Required: true
	  In: query
	*/
	Service string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetPoliciesParams() beforehand.
func (o *GetPoliciesParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qService, qhkService, _ := qs.GetOK("service")
	if err := o.bindService(qService, qhkService, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindService binds and validates parameter Service from query.
func (o *GetPoliciesParams) bindService(rawData []string, hasKey bool, formats strfmt.Registry) error {
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

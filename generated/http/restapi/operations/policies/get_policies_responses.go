// Code generated by go-swagger; DO NOT EDIT.

package policies

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/lunarway/release-manager/generated/http/models"
)

// GetPoliciesOKCode is the HTTP code returned for type GetPoliciesOK
const GetPoliciesOKCode int = 200

/*GetPoliciesOK Found policies

swagger:response getPoliciesOK
*/
type GetPoliciesOK struct {

	/*
	  In: Body
	*/
	Payload *models.GetPoliciesResponse `json:"body,omitempty"`
}

// NewGetPoliciesOK creates GetPoliciesOK with default headers values
func NewGetPoliciesOK() *GetPoliciesOK {

	return &GetPoliciesOK{}
}

// WithPayload adds the payload to the get policies o k response
func (o *GetPoliciesOK) WithPayload(payload *models.GetPoliciesResponse) *GetPoliciesOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get policies o k response
func (o *GetPoliciesOK) SetPayload(payload *models.GetPoliciesResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetPoliciesOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetPoliciesBadRequestCode is the HTTP code returned for type GetPoliciesBadRequest
const GetPoliciesBadRequestCode int = 400

/*GetPoliciesBadRequest Invalid payload

swagger:response getPoliciesBadRequest
*/
type GetPoliciesBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewGetPoliciesBadRequest creates GetPoliciesBadRequest with default headers values
func NewGetPoliciesBadRequest() *GetPoliciesBadRequest {

	return &GetPoliciesBadRequest{}
}

// WithPayload adds the payload to the get policies bad request response
func (o *GetPoliciesBadRequest) WithPayload(payload *models.ErrorResponse) *GetPoliciesBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get policies bad request response
func (o *GetPoliciesBadRequest) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetPoliciesBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetPoliciesNotFoundCode is the HTTP code returned for type GetPoliciesNotFound
const GetPoliciesNotFoundCode int = 404

/*GetPoliciesNotFound Invalid payload

swagger:response getPoliciesNotFound
*/
type GetPoliciesNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewGetPoliciesNotFound creates GetPoliciesNotFound with default headers values
func NewGetPoliciesNotFound() *GetPoliciesNotFound {

	return &GetPoliciesNotFound{}
}

// WithPayload adds the payload to the get policies not found response
func (o *GetPoliciesNotFound) WithPayload(payload *models.ErrorResponse) *GetPoliciesNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get policies not found response
func (o *GetPoliciesNotFound) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetPoliciesNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetPoliciesInternalServerErrorCode is the HTTP code returned for type GetPoliciesInternalServerError
const GetPoliciesInternalServerErrorCode int = 500

/*GetPoliciesInternalServerError Error response

swagger:response getPoliciesInternalServerError
*/
type GetPoliciesInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewGetPoliciesInternalServerError creates GetPoliciesInternalServerError with default headers values
func NewGetPoliciesInternalServerError() *GetPoliciesInternalServerError {

	return &GetPoliciesInternalServerError{}
}

// WithPayload adds the payload to the get policies internal server error response
func (o *GetPoliciesInternalServerError) WithPayload(payload *models.ErrorResponse) *GetPoliciesInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get policies internal server error response
func (o *GetPoliciesInternalServerError) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetPoliciesInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

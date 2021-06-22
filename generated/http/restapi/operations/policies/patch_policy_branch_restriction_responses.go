// Code generated by go-swagger; DO NOT EDIT.

package policies

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/lunarway/release-manager/generated/http/models"
)

// PatchPolicyBranchRestrictionCreatedCode is the HTTP code returned for type PatchPolicyBranchRestrictionCreated
const PatchPolicyBranchRestrictionCreatedCode int = 201

/*PatchPolicyBranchRestrictionCreated Policy applied

swagger:response patchPolicyBranchRestrictionCreated
*/
type PatchPolicyBranchRestrictionCreated struct {

	/*
	  In: Body
	*/
	Payload *models.ApplyBranchRestrictionPolicyResponce `json:"body,omitempty"`
}

// NewPatchPolicyBranchRestrictionCreated creates PatchPolicyBranchRestrictionCreated with default headers values
func NewPatchPolicyBranchRestrictionCreated() *PatchPolicyBranchRestrictionCreated {

	return &PatchPolicyBranchRestrictionCreated{}
}

// WithPayload adds the payload to the patch policy branch restriction created response
func (o *PatchPolicyBranchRestrictionCreated) WithPayload(payload *models.ApplyBranchRestrictionPolicyResponce) *PatchPolicyBranchRestrictionCreated {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch policy branch restriction created response
func (o *PatchPolicyBranchRestrictionCreated) SetPayload(payload *models.ApplyBranchRestrictionPolicyResponce) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchPolicyBranchRestrictionCreated) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(201)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PatchPolicyBranchRestrictionBadRequestCode is the HTTP code returned for type PatchPolicyBranchRestrictionBadRequest
const PatchPolicyBranchRestrictionBadRequestCode int = 400

/*PatchPolicyBranchRestrictionBadRequest Invalid payload

swagger:response patchPolicyBranchRestrictionBadRequest
*/
type PatchPolicyBranchRestrictionBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewPatchPolicyBranchRestrictionBadRequest creates PatchPolicyBranchRestrictionBadRequest with default headers values
func NewPatchPolicyBranchRestrictionBadRequest() *PatchPolicyBranchRestrictionBadRequest {

	return &PatchPolicyBranchRestrictionBadRequest{}
}

// WithPayload adds the payload to the patch policy branch restriction bad request response
func (o *PatchPolicyBranchRestrictionBadRequest) WithPayload(payload *models.ErrorResponse) *PatchPolicyBranchRestrictionBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch policy branch restriction bad request response
func (o *PatchPolicyBranchRestrictionBadRequest) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchPolicyBranchRestrictionBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PatchPolicyBranchRestrictionUnauthorizedCode is the HTTP code returned for type PatchPolicyBranchRestrictionUnauthorized
const PatchPolicyBranchRestrictionUnauthorizedCode int = 401

/*PatchPolicyBranchRestrictionUnauthorized Provided access token was not found or is invalid

swagger:response patchPolicyBranchRestrictionUnauthorized
*/
type PatchPolicyBranchRestrictionUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewPatchPolicyBranchRestrictionUnauthorized creates PatchPolicyBranchRestrictionUnauthorized with default headers values
func NewPatchPolicyBranchRestrictionUnauthorized() *PatchPolicyBranchRestrictionUnauthorized {

	return &PatchPolicyBranchRestrictionUnauthorized{}
}

// WithPayload adds the payload to the patch policy branch restriction unauthorized response
func (o *PatchPolicyBranchRestrictionUnauthorized) WithPayload(payload *models.ErrorResponse) *PatchPolicyBranchRestrictionUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch policy branch restriction unauthorized response
func (o *PatchPolicyBranchRestrictionUnauthorized) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchPolicyBranchRestrictionUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PatchPolicyBranchRestrictionInternalServerErrorCode is the HTTP code returned for type PatchPolicyBranchRestrictionInternalServerError
const PatchPolicyBranchRestrictionInternalServerErrorCode int = 500

/*PatchPolicyBranchRestrictionInternalServerError Error response

swagger:response patchPolicyBranchRestrictionInternalServerError
*/
type PatchPolicyBranchRestrictionInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewPatchPolicyBranchRestrictionInternalServerError creates PatchPolicyBranchRestrictionInternalServerError with default headers values
func NewPatchPolicyBranchRestrictionInternalServerError() *PatchPolicyBranchRestrictionInternalServerError {

	return &PatchPolicyBranchRestrictionInternalServerError{}
}

// WithPayload adds the payload to the patch policy branch restriction internal server error response
func (o *PatchPolicyBranchRestrictionInternalServerError) WithPayload(payload *models.ErrorResponse) *PatchPolicyBranchRestrictionInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch policy branch restriction internal server error response
func (o *PatchPolicyBranchRestrictionInternalServerError) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchPolicyBranchRestrictionInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PatchPolicyBranchRestrictionServiceUnavailableCode is the HTTP code returned for type PatchPolicyBranchRestrictionServiceUnavailable
const PatchPolicyBranchRestrictionServiceUnavailableCode int = 503

/*PatchPolicyBranchRestrictionServiceUnavailable Error response

swagger:response patchPolicyBranchRestrictionServiceUnavailable
*/
type PatchPolicyBranchRestrictionServiceUnavailable struct {

	/*
	  In: Body
	*/
	Payload *models.ErrorResponse `json:"body,omitempty"`
}

// NewPatchPolicyBranchRestrictionServiceUnavailable creates PatchPolicyBranchRestrictionServiceUnavailable with default headers values
func NewPatchPolicyBranchRestrictionServiceUnavailable() *PatchPolicyBranchRestrictionServiceUnavailable {

	return &PatchPolicyBranchRestrictionServiceUnavailable{}
}

// WithPayload adds the payload to the patch policy branch restriction service unavailable response
func (o *PatchPolicyBranchRestrictionServiceUnavailable) WithPayload(payload *models.ErrorResponse) *PatchPolicyBranchRestrictionServiceUnavailable {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch policy branch restriction service unavailable response
func (o *PatchPolicyBranchRestrictionServiceUnavailable) SetPayload(payload *models.ErrorResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchPolicyBranchRestrictionServiceUnavailable) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(503)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
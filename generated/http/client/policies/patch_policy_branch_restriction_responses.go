// Code generated by go-swagger; DO NOT EDIT.

package policies

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/lunarway/release-manager/generated/http/models"
)

// PatchPolicyBranchRestrictionReader is a Reader for the PatchPolicyBranchRestriction structure.
type PatchPolicyBranchRestrictionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PatchPolicyBranchRestrictionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewPatchPolicyBranchRestrictionCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewPatchPolicyBranchRestrictionBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewPatchPolicyBranchRestrictionInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 503:
		result := NewPatchPolicyBranchRestrictionServiceUnavailable()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPatchPolicyBranchRestrictionCreated creates a PatchPolicyBranchRestrictionCreated with default headers values
func NewPatchPolicyBranchRestrictionCreated() *PatchPolicyBranchRestrictionCreated {
	return &PatchPolicyBranchRestrictionCreated{}
}

/* PatchPolicyBranchRestrictionCreated describes a response with status code 201, with default header values.

Policy applied
*/
type PatchPolicyBranchRestrictionCreated struct {
	Payload *models.ApplyBranchRestrictionPolicyResponce
}

func (o *PatchPolicyBranchRestrictionCreated) Error() string {
	return fmt.Sprintf("[PATCH /policy/branch-restriction][%d] patchPolicyBranchRestrictionCreated  %+v", 201, o.Payload)
}
func (o *PatchPolicyBranchRestrictionCreated) GetPayload() *models.ApplyBranchRestrictionPolicyResponce {
	return o.Payload
}

func (o *PatchPolicyBranchRestrictionCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ApplyBranchRestrictionPolicyResponce)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchPolicyBranchRestrictionBadRequest creates a PatchPolicyBranchRestrictionBadRequest with default headers values
func NewPatchPolicyBranchRestrictionBadRequest() *PatchPolicyBranchRestrictionBadRequest {
	return &PatchPolicyBranchRestrictionBadRequest{}
}

/* PatchPolicyBranchRestrictionBadRequest describes a response with status code 400, with default header values.

Invalid payload
*/
type PatchPolicyBranchRestrictionBadRequest struct {
	Payload *models.ErrorResponse
}

func (o *PatchPolicyBranchRestrictionBadRequest) Error() string {
	return fmt.Sprintf("[PATCH /policy/branch-restriction][%d] patchPolicyBranchRestrictionBadRequest  %+v", 400, o.Payload)
}
func (o *PatchPolicyBranchRestrictionBadRequest) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchPolicyBranchRestrictionBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchPolicyBranchRestrictionInternalServerError creates a PatchPolicyBranchRestrictionInternalServerError with default headers values
func NewPatchPolicyBranchRestrictionInternalServerError() *PatchPolicyBranchRestrictionInternalServerError {
	return &PatchPolicyBranchRestrictionInternalServerError{}
}

/* PatchPolicyBranchRestrictionInternalServerError describes a response with status code 500, with default header values.

Error response
*/
type PatchPolicyBranchRestrictionInternalServerError struct {
	Payload *models.ErrorResponse
}

func (o *PatchPolicyBranchRestrictionInternalServerError) Error() string {
	return fmt.Sprintf("[PATCH /policy/branch-restriction][%d] patchPolicyBranchRestrictionInternalServerError  %+v", 500, o.Payload)
}
func (o *PatchPolicyBranchRestrictionInternalServerError) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchPolicyBranchRestrictionInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPatchPolicyBranchRestrictionServiceUnavailable creates a PatchPolicyBranchRestrictionServiceUnavailable with default headers values
func NewPatchPolicyBranchRestrictionServiceUnavailable() *PatchPolicyBranchRestrictionServiceUnavailable {
	return &PatchPolicyBranchRestrictionServiceUnavailable{}
}

/* PatchPolicyBranchRestrictionServiceUnavailable describes a response with status code 503, with default header values.

Error response
*/
type PatchPolicyBranchRestrictionServiceUnavailable struct {
	Payload *models.ErrorResponse
}

func (o *PatchPolicyBranchRestrictionServiceUnavailable) Error() string {
	return fmt.Sprintf("[PATCH /policy/branch-restriction][%d] patchPolicyBranchRestrictionServiceUnavailable  %+v", 503, o.Payload)
}
func (o *PatchPolicyBranchRestrictionServiceUnavailable) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *PatchPolicyBranchRestrictionServiceUnavailable) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

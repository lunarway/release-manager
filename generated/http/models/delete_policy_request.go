// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// DeletePolicyRequest Policies to delete
//
// swagger:model DeletePolicyRequest
type DeletePolicyRequest struct {

	// committer email
	// Required: true
	CommitterEmail *string `json:"committerEmail"`

	// committer name
	// Required: true
	CommitterName *string `json:"committerName"`

	// policy ids
	// Required: true
	PolicyIds []string `json:"policyIds"`

	// service
	// Required: true
	Service *string `json:"service"`
}

// Validate validates this delete policy request
func (m *DeletePolicyRequest) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCommitterEmail(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCommitterName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePolicyIds(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateService(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DeletePolicyRequest) validateCommitterEmail(formats strfmt.Registry) error {

	if err := validate.Required("committerEmail", "body", m.CommitterEmail); err != nil {
		return err
	}

	return nil
}

func (m *DeletePolicyRequest) validateCommitterName(formats strfmt.Registry) error {

	if err := validate.Required("committerName", "body", m.CommitterName); err != nil {
		return err
	}

	return nil
}

func (m *DeletePolicyRequest) validatePolicyIds(formats strfmt.Registry) error {

	if err := validate.Required("policyIds", "body", m.PolicyIds); err != nil {
		return err
	}

	return nil
}

func (m *DeletePolicyRequest) validateService(formats strfmt.Registry) error {

	if err := validate.Required("service", "body", m.Service); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this delete policy request based on context it is used
func (m *DeletePolicyRequest) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *DeletePolicyRequest) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DeletePolicyRequest) UnmarshalBinary(b []byte) error {
	var res DeletePolicyRequest
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

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

// ApplyAutoReleasePolicyRequest Apply an auto-release policy
//
// swagger:model ApplyAutoReleasePolicyRequest
type ApplyAutoReleasePolicyRequest struct {

	// branch
	// Required: true
	Branch *string `json:"branch"`

	// committer email
	// Required: true
	CommitterEmail *string `json:"committerEmail"`

	// committer name
	// Required: true
	CommitterName *string `json:"committerName"`

	// environment
	// Required: true
	Environment *string `json:"environment"`

	// service
	// Required: true
	Service *string `json:"service"`
}

// Validate validates this apply auto release policy request
func (m *ApplyAutoReleasePolicyRequest) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBranch(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCommitterEmail(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCommitterName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateEnvironment(formats); err != nil {
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

func (m *ApplyAutoReleasePolicyRequest) validateBranch(formats strfmt.Registry) error {

	if err := validate.Required("branch", "body", m.Branch); err != nil {
		return err
	}

	return nil
}

func (m *ApplyAutoReleasePolicyRequest) validateCommitterEmail(formats strfmt.Registry) error {

	if err := validate.Required("committerEmail", "body", m.CommitterEmail); err != nil {
		return err
	}

	return nil
}

func (m *ApplyAutoReleasePolicyRequest) validateCommitterName(formats strfmt.Registry) error {

	if err := validate.Required("committerName", "body", m.CommitterName); err != nil {
		return err
	}

	return nil
}

func (m *ApplyAutoReleasePolicyRequest) validateEnvironment(formats strfmt.Registry) error {

	if err := validate.Required("environment", "body", m.Environment); err != nil {
		return err
	}

	return nil
}

func (m *ApplyAutoReleasePolicyRequest) validateService(formats strfmt.Registry) error {

	if err := validate.Required("service", "body", m.Service); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this apply auto release policy request based on context it is used
func (m *ApplyAutoReleasePolicyRequest) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ApplyAutoReleasePolicyRequest) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ApplyAutoReleasePolicyRequest) UnmarshalBinary(b []byte) error {
	var res ApplyAutoReleasePolicyRequest
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

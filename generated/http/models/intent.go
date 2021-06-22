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

// Intent An action intent
//
// swagger:model Intent
type Intent struct {

	// promote
	Promote *IntentPromote `json:"promote,omitempty"`

	// release branch
	ReleaseBranch *IntentReleaseBranch `json:"releaseBranch,omitempty"`

	// rollback
	Rollback *IntentRollback `json:"rollback,omitempty"`

	// type
	// Required: true
	Type *string `json:"type"`
}

// Validate validates this intent
func (m *Intent) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePromote(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateReleaseBranch(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRollback(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Intent) validatePromote(formats strfmt.Registry) error {
	if swag.IsZero(m.Promote) { // not required
		return nil
	}

	if m.Promote != nil {
		if err := m.Promote.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("promote")
			}
			return err
		}
	}

	return nil
}

func (m *Intent) validateReleaseBranch(formats strfmt.Registry) error {
	if swag.IsZero(m.ReleaseBranch) { // not required
		return nil
	}

	if m.ReleaseBranch != nil {
		if err := m.ReleaseBranch.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("releaseBranch")
			}
			return err
		}
	}

	return nil
}

func (m *Intent) validateRollback(formats strfmt.Registry) error {
	if swag.IsZero(m.Rollback) { // not required
		return nil
	}

	if m.Rollback != nil {
		if err := m.Rollback.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("rollback")
			}
			return err
		}
	}

	return nil
}

func (m *Intent) validateType(formats strfmt.Registry) error {

	if err := validate.Required("type", "body", m.Type); err != nil {
		return err
	}

	return nil
}

// ContextValidate validate this intent based on the context it is used
func (m *Intent) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidatePromote(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateReleaseBranch(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateRollback(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Intent) contextValidatePromote(ctx context.Context, formats strfmt.Registry) error {

	if m.Promote != nil {
		if err := m.Promote.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("promote")
			}
			return err
		}
	}

	return nil
}

func (m *Intent) contextValidateReleaseBranch(ctx context.Context, formats strfmt.Registry) error {

	if m.ReleaseBranch != nil {
		if err := m.ReleaseBranch.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("releaseBranch")
			}
			return err
		}
	}

	return nil
}

func (m *Intent) contextValidateRollback(ctx context.Context, formats strfmt.Registry) error {

	if m.Rollback != nil {
		if err := m.Rollback.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("rollback")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Intent) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Intent) UnmarshalBinary(b []byte) error {
	var res Intent
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IntentPromote intent promote
//
// swagger:model IntentPromote
type IntentPromote struct {

	// from environment
	FromEnvironment string `json:"fromEnvironment,omitempty"`
}

// Validate validates this intent promote
func (m *IntentPromote) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this intent promote based on context it is used
func (m *IntentPromote) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IntentPromote) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IntentPromote) UnmarshalBinary(b []byte) error {
	var res IntentPromote
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IntentReleaseBranch intent release branch
//
// swagger:model IntentReleaseBranch
type IntentReleaseBranch struct {

	// branch
	Branch string `json:"branch,omitempty"`
}

// Validate validates this intent release branch
func (m *IntentReleaseBranch) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this intent release branch based on context it is used
func (m *IntentReleaseBranch) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IntentReleaseBranch) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IntentReleaseBranch) UnmarshalBinary(b []byte) error {
	var res IntentReleaseBranch
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// IntentRollback intent rollback
//
// swagger:model IntentRollback
type IntentRollback struct {

	// previous artifact Id
	PreviousArtifactID string `json:"previousArtifactId,omitempty"`
}

// Validate validates this intent rollback
func (m *IntentRollback) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this intent rollback based on context it is used
func (m *IntentRollback) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *IntentRollback) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *IntentRollback) UnmarshalBinary(b []byte) error {
	var res IntentRollback
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

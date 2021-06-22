// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ArtifactStagePush artifact stage push
//
// swagger:model ArtifactStagePush
type ArtifactStagePush struct {

	// data
	Data *ArtifactStagePushData `json:"data,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this artifact stage push
func (m *ArtifactStagePush) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateData(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ArtifactStagePush) validateData(formats strfmt.Registry) error {
	if swag.IsZero(m.Data) { // not required
		return nil
	}

	if m.Data != nil {
		if err := m.Data.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("data")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this artifact stage push based on the context it is used
func (m *ArtifactStagePush) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateData(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ArtifactStagePush) contextValidateData(ctx context.Context, formats strfmt.Registry) error {

	if m.Data != nil {
		if err := m.Data.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("data")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ArtifactStagePush) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ArtifactStagePush) UnmarshalBinary(b []byte) error {
	var res ArtifactStagePush
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// ArtifactStagePushData artifact stage push data
//
// swagger:model ArtifactStagePushData
type ArtifactStagePushData struct {

	// docker version
	DockerVersion string `json:"dockerVersion,omitempty"`

	// image
	Image string `json:"image,omitempty"`

	// tag
	Tag string `json:"tag,omitempty"`
}

// Validate validates this artifact stage push data
func (m *ArtifactStagePushData) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this artifact stage push data based on context it is used
func (m *ArtifactStagePushData) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ArtifactStagePushData) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ArtifactStagePushData) UnmarshalBinary(b []byte) error {
	var res ArtifactStagePushData
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

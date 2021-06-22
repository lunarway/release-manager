// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DescribeArtifactResponse Description of artifacts
//
// swagger:model DescribeArtifactResponse
type DescribeArtifactResponse struct {

	// artifacts
	Artifacts []*Artifact `json:"artifacts"`

	// service
	Service string `json:"service,omitempty"`
}

// Validate validates this describe artifact response
func (m *DescribeArtifactResponse) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateArtifacts(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DescribeArtifactResponse) validateArtifacts(formats strfmt.Registry) error {
	if swag.IsZero(m.Artifacts) { // not required
		return nil
	}

	for i := 0; i < len(m.Artifacts); i++ {
		if swag.IsZero(m.Artifacts[i]) { // not required
			continue
		}

		if m.Artifacts[i] != nil {
			if err := m.Artifacts[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("artifacts" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this describe artifact response based on the context it is used
func (m *DescribeArtifactResponse) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateArtifacts(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DescribeArtifactResponse) contextValidateArtifacts(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Artifacts); i++ {

		if m.Artifacts[i] != nil {
			if err := m.Artifacts[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("artifacts" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *DescribeArtifactResponse) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DescribeArtifactResponse) UnmarshalBinary(b []byte) error {
	var res DescribeArtifactResponse
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
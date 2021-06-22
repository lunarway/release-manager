// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ApplyBranchRestrictionPolicyResponce Branch restriction policy applied
//
// swagger:model ApplyBranchRestrictionPolicyResponce
type ApplyBranchRestrictionPolicyResponce struct {

	// branch regex
	BranchRegex string `json:"branchRegex,omitempty"`

	// environment
	Environment string `json:"environment,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// service
	Service string `json:"service,omitempty"`
}

// Validate validates this apply branch restriction policy responce
func (m *ApplyBranchRestrictionPolicyResponce) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this apply branch restriction policy responce based on context it is used
func (m *ApplyBranchRestrictionPolicyResponce) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ApplyBranchRestrictionPolicyResponce) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ApplyBranchRestrictionPolicyResponce) UnmarshalBinary(b []byte) error {
	var res ApplyBranchRestrictionPolicyResponce
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// EnvironmentStatus environment status
//
// swagger:model EnvironmentStatus
type EnvironmentStatus struct {

	// author
	Author string `json:"author,omitempty"`

	// build Url
	BuildURL string `json:"buildUrl,omitempty"`

	// committer
	Committer string `json:"committer,omitempty"`

	// Epoch timestamp
	Date int64 `json:"date,omitempty"`

	// high vulnerabilities
	HighVulnerabilities int64 `json:"highVulnerabilities,omitempty"`

	// low vulnerabilities
	LowVulnerabilities int64 `json:"lowVulnerabilities,omitempty"`

	// medium vulnerabilities
	MediumVulnerabilities int64 `json:"mediumVulnerabilities,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// tag
	Tag string `json:"tag,omitempty"`
}

// Validate validates this environment status
func (m *EnvironmentStatus) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this environment status based on context it is used
func (m *EnvironmentStatus) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *EnvironmentStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EnvironmentStatus) UnmarshalBinary(b []byte) error {
	var res EnvironmentStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

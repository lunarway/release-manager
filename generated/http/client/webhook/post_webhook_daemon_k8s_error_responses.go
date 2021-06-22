// Code generated by go-swagger; DO NOT EDIT.

package webhook

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/lunarway/release-manager/generated/http/models"
)

// PostWebhookDaemonK8sErrorReader is a Reader for the PostWebhookDaemonK8sError structure.
type PostWebhookDaemonK8sErrorReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PostWebhookDaemonK8sErrorReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPostWebhookDaemonK8sErrorOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPostWebhookDaemonK8sErrorOK creates a PostWebhookDaemonK8sErrorOK with default headers values
func NewPostWebhookDaemonK8sErrorOK() *PostWebhookDaemonK8sErrorOK {
	return &PostWebhookDaemonK8sErrorOK{}
}

/* PostWebhookDaemonK8sErrorOK describes a response with status code 200, with default header values.

OK
*/
type PostWebhookDaemonK8sErrorOK struct {
	Payload models.EmptyWebhookResponse
}

func (o *PostWebhookDaemonK8sErrorOK) Error() string {
	return fmt.Sprintf("[POST /webhook/daemon/k8s/error][%d] postWebhookDaemonK8sErrorOK  %+v", 200, o.Payload)
}
func (o *PostWebhookDaemonK8sErrorOK) GetPayload() models.EmptyWebhookResponse {
	return o.Payload
}

func (o *PostWebhookDaemonK8sErrorOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

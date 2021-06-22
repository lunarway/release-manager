// Code generated by go-swagger; DO NOT EDIT.

package webhook

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/lunarway/release-manager/generated/http/models"
)

// PostWebhookDaemonK8sErrorOKCode is the HTTP code returned for type PostWebhookDaemonK8sErrorOK
const PostWebhookDaemonK8sErrorOKCode int = 200

/*PostWebhookDaemonK8sErrorOK OK

swagger:response postWebhookDaemonK8sErrorOK
*/
type PostWebhookDaemonK8sErrorOK struct {

	/*
	  In: Body
	*/
	Payload models.EmptyWebhookResponse `json:"body,omitempty"`
}

// NewPostWebhookDaemonK8sErrorOK creates PostWebhookDaemonK8sErrorOK with default headers values
func NewPostWebhookDaemonK8sErrorOK() *PostWebhookDaemonK8sErrorOK {

	return &PostWebhookDaemonK8sErrorOK{}
}

// WithPayload adds the payload to the post webhook daemon k8s error o k response
func (o *PostWebhookDaemonK8sErrorOK) WithPayload(payload models.EmptyWebhookResponse) *PostWebhookDaemonK8sErrorOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the post webhook daemon k8s error o k response
func (o *PostWebhookDaemonK8sErrorOK) SetPayload(payload models.EmptyWebhookResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PostWebhookDaemonK8sErrorOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

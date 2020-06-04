package s3storage

import (
	"encoding/json"
)

// S3Event describes events received from S3 notification service.
// They are, kind of, documented here: https://docs.aws.amazon.com/AmazonS3/latest/dev/notification-content-structure.html
type S3Event struct {
	Records []S3EventRecord `json:"Records,omitempty"`
}

func (p *S3Event) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

type S3EventRecord struct {
	EventVersion      string                `json:"eventVersion,omitempty"`
	EventSource       string                `json:"eventSource,omitempty"`
	AwsRegion         string                `json:"awsRegion,omitempty"`
	EventTime         string                `json:"eventTime,omitempty"`
	EventName         string                `json:"eventName,omitempty"`
	UserIdentity      S3EventRecordIdentity `json:"userIdentity,omitempty"`
	RequestParameters map[string]string     `json:"requestParameters,omitempty"`
	ResponseElements  map[string]string     `json:"responseElements,omitempty"`
	S3                S3EventRecordData     `json:"s3,omitempty"`
}

type S3EventRecordIdentity struct {
	PrincipalId string `json:"principalId,omitempty"` //"AWS:AROAIVBPOFACJXCN6JNTG:eki@lunar.app"
}

type S3EventRecordData struct {
	S3SchemaVersion string                  `json:"s3SchemaVersion,omitempty"` //"1.0",
	ConfigurationId string                  `json:"configurationId,omitempty"` //"ZWU1OTJlYTAtOTBhMC00NzkwLWE1NTEtMWE2NGM0MTU5YWRl",
	Bucket          S3EventRecordDataBucket `json:"bucket,omitempty"`
	Object          S3EventRecordDataObject `json:"object,omitempty"`
}

type S3EventRecordDataBucket struct {
	Name          string                `json:"name,omitempty"` //"eki-lunar-artifacts",
	OwnerIdentity S3EventRecordIdentity `json:"ownerIdentity,omitempty"`
	Arn           string                `json:"arn,omitempty"` //"arn:aws:s3:::eki-lunar-artifacts"
}

type S3EventRecordDataObject struct {
	Key       string `json:"key,omitempty"` //"test-service/master-20200604-1303",
	Size      int    `json:"size,omitempty"`
	ETag      string `json:"eTag,omitempty"`      //"c12e30c98ebc8cffb28215a61651614e",
	VersionID string `json:"versionId,omitempty"` // object version if bucket is versioning-enabled, otherwise null
	Sequencer string `json:"sequencer,omitempty"` //"005ED8D51FB571C1E8"
}

module github.com/lunarway/release-manager

go 1.13

require (
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/aws/aws-sdk-go v1.37.11
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/dustin/go-humanize v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/google/uuid v1.2.0
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/lunarway/color v1.7.0
	github.com/makasim/amqpextra v0.14.3
	github.com/manifoldco/promptui v0.8.0
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/nlopes/slack v0.6.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.15.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/streadway/amqp v1.0.1-0.20200716223359-e6b33f460591
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.4.0+incompatible
	go.opencensus.io v0.22.5
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/go-playground/webhooks.v5 v5.17.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.15
	k8s.io/apimachinery v0.18.15
	k8s.io/client-go v0.18.15
	k8s.io/kubernetes v1.18.15
)

// k8s.io/kubernetes has a go.mod file that sets the version of the following
// modules to v0.0.0. This causes go to throw an error. These need to be set
// to a version for Go to process them. Here they are set to the same
// revision as the marked version of Kubernetes. When Kubernetes is updated
// these need to be updated as well.

replace (
	k8s.io/api => k8s.io/api v0.18.15
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.15
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.15
	k8s.io/apiserver => k8s.io/apiserver v0.18.15
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.15
	k8s.io/client-go => k8s.io/client-go v0.18.15
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.15
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.15
	k8s.io/code-generator => k8s.io/code-generator v0.18.15
	k8s.io/component-base => k8s.io/component-base v0.18.15
	k8s.io/cri-api => k8s.io/cri-api v0.18.15
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.15
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.15
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.15
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.15
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.15
	k8s.io/kubectl => k8s.io/kubectl v0.18.15
	k8s.io/kubelet => k8s.io/kubelet v0.18.15
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.15
	k8s.io/metrics => k8s.io/metrics v0.18.15
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.15
)

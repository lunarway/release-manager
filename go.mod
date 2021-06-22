module github.com/lunarway/release-manager

go 1.13

require (
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go v1.37.11
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/dustin/go-humanize v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-openapi/analysis v0.20.1 // indirect
	github.com/go-openapi/errors v0.20.0
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/loads v0.20.2
	github.com/go-openapi/runtime v0.19.29
	github.com/go-openapi/spec v0.20.3
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate v0.20.2
	github.com/google/uuid v1.2.0
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/jessevdk/go-flags v1.5.0
	github.com/johannesboyne/gofakes3 v0.0.0-20210608054100-92d5d4af5fde
	github.com/lunarway/color v1.7.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/makasim/amqpextra v0.16.3
	github.com/manifoldco/promptui v0.8.0
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/nlopes/slack v0.6.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/prometheus/common v0.15.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/streadway/amqp v1.0.1-0.20200716223359-e6b33f460591
	github.com/stretchr/testify v1.7.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.4.0+incompatible
	go.mongodb.org/mongo-driver v1.5.3 // indirect
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
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

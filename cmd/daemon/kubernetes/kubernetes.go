package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	//deploymentutil "k8s.io/kubectl/pkg/util/deployment"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset              *kubernetes.Clientset
	exporter               Exporter
	moduloCrashReportNotif float64
}

var (
	ErrWatcherClosed = errors.New("channel closed")
)

func NewClient(kubeConfigPath string, moduloCrashReportNotif float64, e Exporter) (*Client, error) {
	if kubeConfigPath != "" {
		// we run outside a cluster
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		return &Client{
			clientset: clientset,
		}, nil
	}

	// we run within a cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset:              clientset,
		exporter:               e,
		moduloCrashReportNotif: moduloCrashReportNotif,
	}, nil
}

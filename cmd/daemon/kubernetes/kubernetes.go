package kubernetes

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
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
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
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

func isCorrectlyAnnotated(annotations map[string]string) bool {
	if !(annotations["lunarway.com/controlled-by-release-manager"] == "true") &&
		annotations["lunarway.com/artifact-id"] == "" &&
		annotations["lunarway.com/author"] == "" {
		return false
	}
	return true
}

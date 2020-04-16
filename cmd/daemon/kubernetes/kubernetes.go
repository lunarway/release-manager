package kubernetes

import (
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"


	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset              *kubernetes.Clientset
	exporter               Exporter
	moduloCrashReportNotif float64
	replicaSetTimeDiff     time.Duration
}

var (
	ErrWatcherClosed = errors.New("channel closed")
)

func NewClient(kubeConfigPath string, moduloCrashReportNotif float64, replicaSetTimeDiff time.Duration, e Exporter) (*Client, error) {
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
		replicaSetTimeDiff:     replicaSetTimeDiff,
	}, nil
}

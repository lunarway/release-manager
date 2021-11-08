package kubernetes

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	Clientset       *kubernetes.Clientset
	InformerFactory informers.SharedInformerFactory

	hasSynced chan struct{}
}

var (
	ErrWatcherClosed = errors.New("channel closed")
)

func NewClient(kubeConfigPath string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)

	return &Client{
		Clientset:       clientset,
		InformerFactory: factory,

		hasSynced: make(chan struct{}),
	}, nil
}

func (c *Client) Start(stopCh chan struct{}) error {
	c.InformerFactory.Start(stopCh)

	syncStatus := c.InformerFactory.WaitForCacheSync(stopCh)
	for informer, synced := range syncStatus {
		if !synced {
			return fmt.Errorf("failed to sync informer '%v'", informer)
		}
		log.Infof("Synced informer '%v'", informer)
	}

	close(c.hasSynced)

	return nil
}

func (c *Client) HasSynced() bool {
	select {
	case <-c.hasSynced:
		return true
	default:
		return false
	}
}

func isCorrectlyAnnotated(annotations map[string]string) bool {
	if (annotations["lunarway.com/controlled-by-release-manager"] == "true") &&
		annotations["lunarway.com/artifact-id"] != "" &&
		annotations["lunarway.com/author"] != "" {
		return true
	}
	return false
}

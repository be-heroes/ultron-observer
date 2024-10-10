package kubernetes

import (
	"fmt"

	mapper "github.com/be-heroes/ultron/pkg/mapper"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type IKubernetesClient interface {
}

type KubernetesClient struct {
	client               *kubernetes.Clientset
	mapper               *mapper.IMapper
	kubernetesMasterUrl  string
	kubernetesConfigPath string
}

func NewKubernetesClient(kubernetesMasterUrl string, kubernetesConfigPath string, mapper *mapper.IMapper) (*KubernetesClient, error) {
	var err error

	if kubernetesMasterUrl == "tcp://:" {
		kubernetesMasterUrl = ""
	}

	config, err := clientcmd.BuildConfigFromFlags(kubernetesMasterUrl, kubernetesConfigPath)
	if err != nil {
		fmt.Println("Falling back to docker Kubernetes API at  https://kubernetes.docker.internal:6443")

		config = &rest.Config{
			Host: "https://kubernetes.docker.internal:6443",
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesClient{
		client:               clientset,
		kubernetesMasterUrl:  kubernetesMasterUrl,
		kubernetesConfigPath: kubernetesConfigPath,
		mapper:               mapper,
	}, nil
}

package kubernetes

import (
	"context"
	"fmt"
	"strconv"

	mapper "github.com/be-heroes/ultron/pkg/mapper"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type IKubernetesClient interface {
	GetNodeMetrics() (map[string]map[string]string, error)
	GetPodMetrics() (map[string]map[string]string, error)
}

type KubernetesClient struct {
	config               *rest.Config
	mapper               mapper.IMapper
	kubernetesMasterUrl  string
	kubernetesConfigPath string
}

func NewKubernetesClient(kubernetesMasterUrl string, kubernetesConfigPath string, mapper mapper.IMapper) (*KubernetesClient, error) {
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

	return &KubernetesClient{
		config:               config,
		kubernetesMasterUrl:  kubernetesMasterUrl,
		kubernetesConfigPath: kubernetesConfigPath,
		mapper:               mapper,
	}, nil
}

func (k *KubernetesClient) GetNodeMetrics() (map[string]map[string]string, error) {
	metricsClient, err := metricsclient.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]map[string]string)
	for _, nodeMetric := range nodeMetrics.Items {
		cpuUsage := nodeMetric.Usage["cpu"]
		memoryUsage := nodeMetric.Usage["memory"]

		metrics[nodeMetric.Name] = map[string]string{
			"cpuUsage":    cpuUsage.AsDec().String(),
			"memoryUsage": memoryUsage.AsDec().String(),
		}
	}

	return metrics, nil
}

func (k *KubernetesClient) GetPodMetrics() (map[string]map[string]string, error) {
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	metricsClient, err := metricsclient.NewForConfig(k.config)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]map[string]string)
	for _, namespace := range namespaces.Items {
		podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses(namespace.Name).List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			return nil, err
		}

		for _, podMetric := range podMetricsList.Items {
			cpuTotal := int64(0)
			memoryTotal := int64(0)

			for _, container := range podMetric.Containers {
				cpuUsage := container.Usage[corev1.ResourceCPU]
				memUsage := container.Usage[corev1.ResourceMemory]

				cpuTotal += cpuUsage.MilliValue()
				memoryTotal += memUsage.Value()
			}

			metrics[podMetric.Name] = map[string]string{
				"cpuTotal":    strconv.FormatInt(cpuTotal, 10),
				"memoryTotal": strconv.FormatInt(memoryTotal, 10),
			}
		}
	}

	return metrics, nil
}

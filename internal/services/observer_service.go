package services

import (
	"github.com/be-heroes/ultron-observer/internal/clients/kubernetes"
	mapper "github.com/be-heroes/ultron/pkg/mapper"
	corev1 "k8s.io/api/core/v1"
)

type IObserverService interface {
	ObservePod(pod *corev1.Pod) error
	ObserveNode(node *corev1.Node) error
}

type ObserverService struct {
	client *kubernetes.IKubernetesClient
	mapper *mapper.IMapper
}

func NewObserverService(client *kubernetes.IKubernetesClient, mapper *mapper.IMapper) *ObserverService {
	return &ObserverService{
		client: client,
		mapper: mapper,
	}
}

func (o *ObserverService) ObservePod(pod *corev1.Pod) error {
	// TODO: Impl logic to observe Pod

	return nil
}

func (o *ObserverService) ObserveNode(node *corev1.Node) error {
	// TODO: Impl logic to observe Node

	return nil
}

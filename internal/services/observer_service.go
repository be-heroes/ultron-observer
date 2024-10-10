package services

import (
	"context"
	"errors"

	"github.com/be-heroes/ultron-observer/internal/clients/kubernetes"
	mapper "github.com/be-heroes/ultron/pkg/mapper"
	corev1 "k8s.io/api/core/v1"
)

type IObserverService interface {
	ObservePod(ctx context.Context, pod *corev1.Pod, errChan chan<- error)
	ObserveNode(ctx context.Context, node *corev1.Node, errChan chan<- error)
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

func (o *ObserverService) ObservePod(ctx context.Context, pod *corev1.Pod, errChan chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("goroutine canceled")
		default:
			// TODO: Impl logic to observe Pod
			errChan <- nil
		}
	}
}

func (o *ObserverService) ObserveNode(ctx context.Context, node *corev1.Node, errChan chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("goroutine canceled")
		default:
			// TODO: Impl logic to observe Node
			errChan <- nil
		}
	}
}

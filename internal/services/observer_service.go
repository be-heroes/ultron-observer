package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	observer "github.com/be-heroes/ultron-observer/pkg"
	mapper "github.com/be-heroes/ultron/pkg/mapper"
	services "github.com/be-heroes/ultron/pkg/services"
	"github.com/redis/go-redis/v9"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IObserverService interface {
	ObservePod(ctx context.Context, pod *corev1.Pod, errChan chan<- error)
	ObserveNode(ctx context.Context, node *corev1.Node, errChan chan<- error)
}

type ObserverService struct {
	kubernetesService services.IKubernetesService
	redisClient       *redis.Client
	mapper            mapper.IMapper
}

func NewObserverService(kubernetesService services.IKubernetesService, redisClient *redis.Client, mapper mapper.IMapper) *ObserverService {
	return &ObserverService{
		kubernetesService: kubernetesService,
		redisClient:       redisClient,
		mapper:            mapper,
	}
}

func (o *ObserverService) ObservePod(ctx context.Context, pod *corev1.Pod, errChan chan<- error) {
	podKey := fmt.Sprintf("%s:%s:%s", observer.CacheKeyPrefixPod, pod.Namespace, pod.Name)

	exists, err := o.redisClient.Exists(ctx, podKey).Result()
	if err != nil {
		errChan <- fmt.Errorf("error checking Redis cache: %w", err)

		return
	}

	if exists > 0 {
		errChan <- errors.New("pod is already being observed")

		return
	}

	err = o.redisClient.Set(ctx, podKey, "observing", 0).Err()
	if err != nil {
		errChan <- fmt.Errorf("error setting Redis cache: %w", err)

		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()

		o.redisClient.Del(ctx, podKey)
	}()

	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("observation canceled")

			return
		case <-ticker.C:
			exists, err := o.kubernetesService.GetPods(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", pod.Name),
			})
			if err != nil || exists == nil {
				errChan <- fmt.Errorf("error fetching pod: %w", err)

				return
			}

			_, err = o.kubernetesService.GetPodMetrics(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", pod.Name),
			})
			if err != nil {
				errChan <- fmt.Errorf("error fetching pod metrics: %w", err)
				continue
			}

			//TODO: Convert pod metrics to OTLP format and send to OpenTelemetry Collector.

			errChan <- nil
		}
	}
}

func (o *ObserverService) ObserveNode(ctx context.Context, node *corev1.Node, errChan chan<- error) {
	nodeKey := fmt.Sprintf("%s:%s", observer.CacheKeyPrefixNode, node.Name)

	exists, err := o.redisClient.Exists(ctx, nodeKey).Result()
	if err != nil {
		errChan <- fmt.Errorf("error checking Redis cache: %w", err)

		return
	}

	if exists > 0 {
		errChan <- errors.New("node is already being observed")

		return
	}

	err = o.redisClient.Set(ctx, nodeKey, "observing", 0).Err()

	if err != nil {
		errChan <- fmt.Errorf("error setting Redis cache: %w", err)

		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()

		o.redisClient.Del(ctx, nodeKey)
	}()

	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("observation canceled")

			return
		case <-ticker.C:
			exists, err := o.kubernetesService.GetNodes(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", node.Name),
			})
			if err != nil || exists == nil {
				errChan <- fmt.Errorf("error fetching node: %w", err)

				return
			}

			_, err = o.kubernetesService.GetNodeMetrics(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", node.Name),
			})
			if err != nil {
				errChan <- fmt.Errorf("error fetching node metrics: %w", err)
				continue
			}

			//TODO: Convert node metrics to OTLP format and send to OpenTelemetry Collector.

			errChan <- nil
		}
	}
}

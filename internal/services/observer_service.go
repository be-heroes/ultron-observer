package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	otlp "github.com/be-heroes/ultron-observer/internal/clients/otlp"
	observer "github.com/be-heroes/ultron-observer/pkg"
	ultron "github.com/be-heroes/ultron/pkg"
	mapper "github.com/be-heroes/ultron/pkg/mapper"
	services "github.com/be-heroes/ultron/pkg/services"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IObserverService interface {
	ObservePod(ctx context.Context, wPod *ultron.WeightedPod, errChan chan<- error)
	ObserveNode(ctx context.Context, wNode *ultron.WeightedNode, errChan chan<- error)
}

type ObserverService struct {
	kubernetesService services.IKubernetesService
	redisClient       *redis.Client
	meterClient       otlp.IMeterClient
	mapper            mapper.IMapper
}

func NewObserverService(kubernetesService services.IKubernetesService, redisClient *redis.Client, meterClient otlp.IMeterClient, mapper mapper.IMapper) (*ObserverService, error) {
	meterProvider, err := meterClient.GetMeterProvider(context.Background())
	if err != nil {
		return nil, err
	}

	otel.SetMeterProvider(meterProvider)

	return &ObserverService{
		kubernetesService: kubernetesService,
		redisClient:       redisClient,
		meterClient:       meterClient,
		mapper:            mapper,
	}, nil
}

func (o *ObserverService) ObservePod(ctx context.Context, wPod *ultron.WeightedPod, errChan chan<- error) {
	podKey := fmt.Sprintf("%s:%s", observer.CacheKeyPrefixPod, mapToString(wPod.Selector))

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

	meter := otel.Meter(observer.MeterKeyPod)

	cpuCounter, err := meter.Int64Counter(ultron.WeightKeyCpuTotal)
	if err != nil {
		errChan <- fmt.Errorf("error creating cpu counter: %w", err)
		return
	}

	memoryCounter, err := meter.Int64Counter(ultron.WeightKeyMemoryTotal)
	if err != nil {
		errChan <- fmt.Errorf("error creating memory counter: %w", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("observation canceled")

			return
		case <-ticker.C:
			exists, err := o.kubernetesService.GetPods(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("%s=%s", ultron.MetadataName, wPod.Selector[ultron.MetadataName]),
			})
			if err != nil || exists == nil {
				errChan <- fmt.Errorf("error fetching pod: %w", err)

				return
			}

			podMetrics, err := o.kubernetesService.GetPodMetrics(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("%s=%s", ultron.MetadataName, wPod.Selector[ultron.MetadataName]),
			})
			if err != nil {
				errChan <- fmt.Errorf("error fetching pod metrics: %w", err)
				continue
			}

			for _, value := range podMetrics {
				cpuTotal, err := strconv.Atoi(value[ultron.WeightKeyCpuTotal])
				if err != nil {
					errChan <- fmt.Errorf("error parsing CPU total: %w", err)
					continue
				}

				memoryTotal, err := strconv.Atoi(value[ultron.WeightKeyMemoryTotal])
				if err != nil {
					errChan <- fmt.Errorf("error parsing memory total: %w", err)
					continue
				}

				cpuCounter.Add(ctx, int64(cpuTotal), metric.WithAttributes(attribute.String("pod", wPod.Selector[ultron.MetadataName])))
				memoryCounter.Add(ctx, int64(memoryTotal), metric.WithAttributes(attribute.String("pod", wPod.Selector[ultron.MetadataName])))
			}

			errChan <- nil
		}
	}
}

func (o *ObserverService) ObserveNode(ctx context.Context, wNode *ultron.WeightedNode, errChan chan<- error) {
	nodeKey := fmt.Sprintf("%s:%s", observer.CacheKeyPrefixNode, mapToString(wNode.Selector))

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

	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()

		o.redisClient.Del(ctx, nodeKey)
	}()

	meter := otel.Meter(observer.MeterKeyNode)

	cpuCounter, err := meter.Int64Counter(ultron.WeightKeyCpuUsage)
	if err != nil {
		errChan <- fmt.Errorf("error creating cpu counter: %w", err)
		return
	}

	memoryCounter, err := meter.Int64Counter(ultron.WeightKeyMemoryUsage)
	if err != nil {
		errChan <- fmt.Errorf("error creating memory counter: %w", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			errChan <- errors.New("observation canceled")

			return
		case <-ticker.C:
			exists, err := o.kubernetesService.GetNodes(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("%s=%s", ultron.MetadataName, wNode.Selector[ultron.LabelHostName]),
			})
			if err != nil || exists == nil {
				errChan <- fmt.Errorf("error fetching node: %w", err)

				return
			}

			nodeMetrics, err := o.kubernetesService.GetNodeMetrics(ctx, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("%s=%s", ultron.MetadataName, wNode.Selector[ultron.LabelHostName]),
			})
			if err != nil {
				errChan <- fmt.Errorf("error fetching node metrics: %w", err)
				continue
			}

			for _, value := range nodeMetrics {
				cpuUsage, err := strconv.Atoi(value[ultron.WeightKeyCpuUsage])
				if err != nil {
					errChan <- fmt.Errorf("error parsing CPU usage: %w", err)
					continue
				}

				memoryUsage, err := strconv.Atoi(value[ultron.WeightKeyMemoryUsage])
				if err != nil {
					errChan <- fmt.Errorf("error parsing memory usage: %w", err)
					continue
				}

				cpuCounter.Add(ctx, int64(cpuUsage), metric.WithAttributes(attribute.String("node", wNode.Selector[ultron.LabelHostName])))
				memoryCounter.Add(ctx, int64(memoryUsage), metric.WithAttributes(attribute.String("node", wNode.Selector[ultron.LabelHostName])))
			}

			errChan <- nil
		}
	}
}

func mapToString(m map[string]string) string {
	var sb strings.Builder

	for key, value := range m {
		sb.WriteString(key + "=" + value + ", ")
	}

	result := sb.String()

	// Trim the trailing comma and space
	if len(result) > 2 {
		result = result[:len(result)-2]
	}

	return result
}

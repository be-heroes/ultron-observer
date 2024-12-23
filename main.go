package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"

	otlp "github.com/be-heroes/ultron-observer/internal/clients/otlp"
	services "github.com/be-heroes/ultron-observer/internal/services"
	observer "github.com/be-heroes/ultron-observer/pkg"
	ultron "github.com/be-heroes/ultron/pkg"
	"github.com/be-heroes/ultron/pkg/mapper"

	"github.com/redis/go-redis/v9"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Info("Initializing ultron-observer")

	config, err := observer.LoadConfig()
	if err != nil {
		sugar.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdown
		sugar.Info("Received shutdown signal, stopping observer...")
		cancel()
	}()

	redisClient := ultron.InitializeRedisClient(config.RedisServerAddress, config.RedisServerPassword, config.RedisServerDatabase)
	if redisClient != nil {
		if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
			sugar.Fatalw("Failed to connect to Redis", "error", err)
		}
	}

	kubernetesClient, err := observer.InitializeKubernetesServiceFromConfig(config)
	if err != nil {
		sugar.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	meterClient := otlp.NewMeterClient("config.OTLPMetricsCollectorURL", "config.ServiceName")

	mapper := mapper.NewMapper()
	observer, err := services.NewObserverService(kubernetesClient, redisClient, meterClient, mapper)

	if err != nil {
		sugar.Fatalf("Failed to initialize observer service: %v", err)
	}

	sugar.Info("Initialized ultron-observer")
	sugar.Info("Starting ultron-observer")

	var wg sync.WaitGroup

	if redisClient != nil {
		pubsub := redisClient.Subscribe(ctx, ultron.TopicPodObserve, ultron.TopicNodeObserve)
		defer pubsub.Close()

		ch := pubsub.Channel()
		sugar.Info("Subscribed to Redis channels. Waiting for messages...")

		for {
			select {
			case <-ctx.Done():
				sugar.Info("Context cancelled, stopping message processing loop.")

				wg.Wait()

				sugar.Info("All goroutines completed, exiting.")

				return
			case msg := <-ch:
				if msg == nil {
					continue
				}

				wg.Add(1)

				go func(msg *redis.Message) {
					defer wg.Done()

					processMessage(ctx, observer, msg, sugar)
				}(msg)
			}
		}
	}
}

func processMessage(ctx context.Context, observer services.IObserverService, msg *redis.Message, sugar *zap.SugaredLogger) {
	switch msg.Channel {
	case ultron.TopicPodObserve:
		var wPod ultron.WeightedPod

		if err := json.Unmarshal([]byte(msg.Payload), &wPod); err != nil {
			sugar.Errorf("Error deserializing msg.Payload to Pod: %v", err)
			return
		}

		errChan := make(chan error, 1)

		go observer.ObservePod(ctx, &wPod, errChan)

		if err := <-errChan; err != nil {
			sugar.Errorf("Error occurred while observing Pod: %v", err)
		}

	case ultron.TopicNodeObserve:
		var wNode ultron.WeightedNode

		if err := json.Unmarshal([]byte(msg.Payload), &wNode); err != nil {
			sugar.Errorf("Error deserializing msg.Payload to Node: %v", err)
			return
		}

		errChan := make(chan error, 1)

		go observer.ObserveNode(ctx, &wNode, errChan)

		if err := <-errChan; err != nil {
			sugar.Errorf("Error occurred while observing Node: %v", err)
		}

	default:
		sugar.Warnf("Received message from unsupported channel: %s", msg.Channel)
	}
}

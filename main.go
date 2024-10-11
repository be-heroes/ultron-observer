package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"go.uber.org/zap"

	attendant "github.com/be-heroes/ultron-attendant/pkg"
	"github.com/be-heroes/ultron-observer/internal/clients/kubernetes"
	"github.com/be-heroes/ultron-observer/internal/services"
	ultron "github.com/be-heroes/ultron/pkg"
	"github.com/be-heroes/ultron/pkg/mapper"

	"github.com/redis/go-redis/v9"
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	RedisServerAddress   string
	RedisServerPassword  string
	RedisServerDatabase  int
	KubernetesConfigPath string
	KubernetesMasterURL  string
}

func LoadConfig() (*Config, error) {
	redisDatabase, err := strconv.Atoi(os.Getenv(ultron.EnvRedisServerDatabase))
	if err != nil {
		redisDatabase = 0
	}

	return &Config{
		RedisServerAddress:   os.Getenv(ultron.EnvRedisServerAddress),
		RedisServerPassword:  os.Getenv(ultron.EnvRedisServerPassword),
		RedisServerDatabase:  redisDatabase,
		KubernetesConfigPath: os.Getenv(attendant.EnvKubernetesConfig),
		KubernetesMasterURL:  fmt.Sprintf("tcp://%s:%s", os.Getenv(attendant.EnvKubernetesServiceHost), os.Getenv(attendant.EnvKubernetesServicePort)),
	}, nil
}

func initializeRedisClient(ctx context.Context, config *Config, sugar *zap.SugaredLogger) *redis.Client {
	if config.RedisServerAddress == "" {
		return nil
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisServerAddress,
		Password: config.RedisServerPassword,
		DB:       config.RedisServerDatabase,
	})

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		sugar.Fatalf("Failed to ping redis server with error: %v", err)
	}

	return redisClient
}

func initializeKubernetesClient(config *Config, mapper mapper.IMapper, sugar *zap.SugaredLogger) kubernetes.IKubernetesClient {
	kubernetesClient, err := kubernetes.NewKubernetesClient(config.KubernetesMasterURL, config.KubernetesConfigPath, mapper)

	if err != nil {
		sugar.Fatalf("Failed to initialize kubernetes client with error: %v", err)
	}

	return kubernetesClient
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Info("Initializing ultron-observer")

	config, err := LoadConfig()
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

	mapperInstance := mapper.NewMapper()
	redisClient := initializeRedisClient(ctx, config, sugar)
	kubernetesClient := initializeKubernetesClient(config, mapperInstance, sugar)
	observer := services.NewObserverService(kubernetesClient, redisClient, mapperInstance)

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
		var pod corev1.Pod

		if err := json.Unmarshal([]byte(msg.Payload), &pod); err != nil {
			sugar.Errorf("Error deserializing msg.Payload to Pod: %v", err)
			return
		}

		errChan := make(chan error, 1)

		go observer.ObservePod(ctx, &pod, errChan)

		if err := <-errChan; err != nil {
			sugar.Errorf("Error occurred while observing Pod: %v", err)
		}

	case ultron.TopicNodeObserve:
		var node corev1.Node

		if err := json.Unmarshal([]byte(msg.Payload), &node); err != nil {
			sugar.Errorf("Error deserializing msg.Payload to Node: %v", err)
			return
		}

		errChan := make(chan error, 1)

		go observer.ObserveNode(ctx, &node, errChan)

		if err := <-errChan; err != nil {
			sugar.Errorf("Error occurred while observing Node: %v", err)
		}

	default:
		sugar.Warnf("Received message from unsupported channel: %s", msg.Channel)
	}
}

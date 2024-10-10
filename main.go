package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	attendant "github.com/be-heroes/ultron-attendant/pkg"
	"github.com/be-heroes/ultron-observer/internal/clients/kubernetes"
	"github.com/be-heroes/ultron-observer/internal/services"
	ultron "github.com/be-heroes/ultron/pkg"
	"github.com/be-heroes/ultron/pkg/mapper"

	"github.com/redis/go-redis/v9"
	corev1 "k8s.io/api/core/v1"
)

func main() {
	log.Println("Initializing ultron-observer")

	var redisClient *redis.Client

	redisServerAddress := os.Getenv(ultron.EnvRedisServerAddress)
	redisServerDatabase := os.Getenv(ultron.EnvRedisServerDatabase)
	redisServerDatabaseInt, err := strconv.Atoi(redisServerDatabase)
	if err != nil {
		redisServerDatabaseInt = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		log.Println("Received shutdown signal, stopping observer...")
		cancel()
	}()

	if redisServerAddress != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisServerAddress,
			Password: os.Getenv(ultron.EnvRedisServerPassword),
			DB:       redisServerDatabaseInt,
		})

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Failed to ping redis server with error: %v", err)
		}
	}

	var kubernetesClient kubernetes.IKubernetesClient
	var observer services.IObserverService
	var mapper mapper.IMapper = mapper.NewMapper()

	kubernetesConfigPath := os.Getenv(attendant.EnvKubernetesConfig)
	kubernetesMasterUrl := fmt.Sprintf("tcp://%s:%s", os.Getenv(attendant.EnvKubernetesServiceHost), os.Getenv(attendant.EnvKubernetesServicePort))
	kubernetesClient, err = kubernetes.NewKubernetesClient(kubernetesMasterUrl, kubernetesConfigPath, mapper)
	if err != nil {
		log.Fatalf("Failed to initialize kubernetes client with error: %v", err)
	}

	observer = services.NewObserverService(&kubernetesClient, &mapper)

	log.Println("Initialized ultron-observer")
	log.Println("Starting ultron-observer")

	var wg sync.WaitGroup

	if redisClient != nil {
		pubsub := redisClient.Subscribe(ctx, ultron.TopicPodObserve, ultron.TopicNodeObserve)
		defer pubsub.Close()

		ch := pubsub.Channel()
		log.Println("Subscribed to Redis channels. Waiting for messages...")

		for {
			select {
			case <-ctx.Done():
				log.Println("Context cancelled, stopping message processing loop.")
				wg.Wait()
				log.Println("All goroutines completed, exiting.")
				return
			case msg := <-ch:
				if msg == nil {
					continue
				}

				wg.Add(1)
				go func(msg *redis.Message) {
					defer wg.Done()
					processMessage(ctx, observer, msg)
				}(msg)
			}
		}
	}
}

func processMessage(ctx context.Context, observer services.IObserverService, msg *redis.Message) {
	switch msg.Channel {
	case ultron.TopicPodObserve:
		var pod corev1.Pod
		if err := json.Unmarshal([]byte(msg.Payload), &pod); err != nil {
			log.Printf("Error deserializing msg.Payload to Pod: %v", err)
			return
		}
		errChan := make(chan error, 1)
		go observer.ObservePod(ctx, &pod, errChan)

		if err := <-errChan; err != nil {
			log.Printf("Error occurred while observing Pod: %v", err)
		}

	case ultron.TopicNodeObserve:
		var node corev1.Node
		if err := json.Unmarshal([]byte(msg.Payload), &node); err != nil {
			log.Printf("Error deserializing msg.Payload to Node: %v", err)
			return
		}
		errChan := make(chan error, 1)
		go observer.ObserveNode(ctx, &node, errChan)

		if err := <-errChan; err != nil {
			log.Printf("Error occurred while observing Node: %v", err)
		}

	default:
		log.Printf("Received message from unsupported channel: %s", msg.Channel)
	}
}

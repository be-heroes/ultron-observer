package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	attendant "github.com/be-heroes/ultron-attendant/pkg"
	"github.com/be-heroes/ultron-observer/internal/clients/kubernetes"
	services "github.com/be-heroes/ultron-observer/internal/services"
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

	ctx := context.Background()

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

	errChannels := make([]chan error, 0)
	kubernetesConfigPath := os.Getenv(attendant.EnvKubernetesConfig)
	kubernetesMasterUrl := fmt.Sprintf("tcp://%s:%s", os.Getenv(attendant.EnvKubernetesServiceHost), os.Getenv(attendant.EnvKubernetesServicePort))
	kubernetesClient, err = kubernetes.NewKubernetesClient(kubernetesMasterUrl, kubernetesConfigPath, &mapper)
	if err != nil {
		log.Fatalf("Failed to initialize kubernetes client with error: %v", err)
	}

	observer = services.NewObserverService(&kubernetesClient, &mapper)

	log.Println("Initialized ultron-observer")
	log.Println("Starting ultron-observer")

	if redisClient != nil {
		pubsub := redisClient.Subscribe(ctx, ultron.TopicPodObserve, ultron.TopicNodeObserve)
		defer pubsub.Close()

		ch := pubsub.Channel()

		fmt.Println("Subscribed to redis channels. Waiting for messages...")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for msg := range ch {
			switch msg.Channel {
			case ultron.TopicPodObserve:
				var pod corev1.Pod

				err := json.Unmarshal([]byte(msg.Payload), &pod)
				if err != nil {
					log.Fatalf("Error deserializing msg.Payload to Pod: %v", err)
				}

				errChan := make(chan error)

				go observer.ObservePod(ctx, &pod, errChan)

				errChannels = append(errChannels, errChan)
			case ultron.TopicNodeObserve:
				var node corev1.Node

				err := json.Unmarshal([]byte(msg.Payload), &node)
				if err != nil {
					log.Fatalf("Error deserializing msg.Payload to Node: %v", err)
				}

				errChan := make(chan error)

				go observer.ObserveNode(ctx, &node, errChan)

				errChannels = append(errChannels, errChan)
			default:
				fmt.Printf("Received message from unsupported channel: %s\n", msg.Channel)
			}
		}
	}

	for _, errChan := range errChannels {
		err := <-errChan
		if err != nil {
			log.Fatalf("Error occurred while observing: %v", err)
		}
	}

	log.Println("Stopped ultron-observer")
}

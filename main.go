package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	ultron "github.com/be-heroes/ultron/pkg"

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

	log.Println("Initialized ultron-observer")
	log.Println("Starting ultron-observer")

	if redisClient != nil {
		pubsub := redisClient.Subscribe(ctx, ultron.TopicPodObserve, ultron.TopicNodeObserve)
		defer pubsub.Close()

		ch := pubsub.Channel()

		fmt.Println("Subscribed to redis channels. Waiting for messages...")

		for msg := range ch {
			switch msg.Channel {
			case ultron.TopicPodObserve:
				var pod corev1.Pod

				err := json.Unmarshal([]byte(msg.Payload), &pod)
				if err != nil {
					log.Fatalf("Error deserializing msg.Payload to Pod: %v", err)
				}

				// TODO: Implement logic to handle Pod events
			case ultron.TopicNodeObserve:
				var node corev1.Node

				err := json.Unmarshal([]byte(msg.Payload), &node)
				if err != nil {
					log.Fatalf("Error deserializing msg.Payload to Node: %v", err)
				}

				// TODO: Implement logic to handle Node events
			default:
				fmt.Printf("Received message from unsupported channel: %s\n", msg.Channel)
			}
		}
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	ultron "github.com/be-heroes/ultron/pkg"

	"github.com/redis/go-redis/v9"
)

func main() {
	var redisClient *redis.Client

	redisServerAddress := os.Getenv(ultron.EnvRedisServerAddress)
	redisServerDatabase := os.Getenv(ultron.EnvRedisServerDatabase)
	redisServerDatabaseInt, err := strconv.Atoi(redisServerDatabase)
	if err != nil {
		redisServerDatabaseInt = 0
	}

	if redisServerAddress != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisServerAddress,
			Password: os.Getenv(ultron.EnvRedisServerPassword),
			DB:       redisServerDatabaseInt,
		})

		ctx := context.Background()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Fatalf("Failed to ping redis server with error: %v", err)
		}

		pubsub := redisClient.Subscribe(ctx, ultron.TopicPodObserve, ultron.TopicNodeObserve)
		defer pubsub.Close()

		ch := pubsub.Channel()

		fmt.Println("Subscribed to channels. Waiting for messages...")

		for msg := range ch {
			fmt.Printf("Received message from %s: %s\n", msg.Channel, msg.Payload)
		}
	}
}

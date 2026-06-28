package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient initializes a new Redis connection
func NewRedisClient(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Ping to test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Redis")
	return client, nil
}

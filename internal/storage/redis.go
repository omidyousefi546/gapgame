package storage

import (
	"context"
	"github.com/redis/go-redis/v9"
)

func NewRedis(url string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: url,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var Client *redis.Client

func Init(redisAddr string, redisPassword string, defaultDB int) {
	Client = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       defaultDB,
	})
	err := Client.Set(ctx, "key", "value", 30).Err()
	if err != nil {
		panic(err)
	}
	_, err = Client.Get(ctx, "key2").Result()
	if err != redis.Nil && err != nil {
		panic(err)
	}
}

func Get(key string) string {
	val, err := Client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return val
}

func Set(key string, value string) error {
	return Client.Set(ctx, key, value, 0).Err()
}

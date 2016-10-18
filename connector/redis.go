package connector

import (
	"gopkg.in/redis.v5"
)

var client *redis.Client

func GetRedis() *redis.Client {
	if client == nil {
		client = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
	}
	return client
}

package util

import (
	"gopkg.in/redis.v5"
	"github.com/cihub/seelog"
)

var oRedisClient *redis.Client

func GetRedis() *redis.Client {
	if oRedisClient == nil {
		oRedisClient = redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		})
		_, err := oRedisClient.Ping().Result()
		if (err != nil) {
			seelog.Error("Init Redis Error")
		}
	}
	return oRedisClient
}

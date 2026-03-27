package redis

import (
	"einvoice-access-point/pkg/config"
	"log"

	"github.com/go-redis/redis/v8"
)

type Redis struct {
	Red *redis.Client
}

func NewRedisConnection(rdb *redis.Client) *Redis {
	return &Redis{Red: rdb}
}

func (rdb *Redis) RedisDb() *redis.Client {
	return rdb.Red
}

func NewClient() *redis.Client {
	redisConfig := config.GetConfig().Redis
	log.Println(redisConfig.REDIS_URL)
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.REDIS_URL,
		Password: "",
		DB:       0,
	})
	return client
}

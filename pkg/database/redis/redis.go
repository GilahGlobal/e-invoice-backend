package redis

import (
	"einvoice-access-point/pkg/config"

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
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.REDIS_HOST + ":" + redisConfig.REDIS_PORT,
		Password: "",
		DB:       0,
	})
	return client
}

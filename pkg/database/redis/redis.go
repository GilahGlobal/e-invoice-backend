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

	log.Println("Connecting to Redis:", redisConfig.REDIS_URL)

	// ✅ Proper parsing (handles TLS, auth, etc)
	opt, err := redis.ParseURL(redisConfig.REDIS_URL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	return client
}

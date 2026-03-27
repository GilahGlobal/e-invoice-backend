/*package redis

import (
	"einvoice-access-point/pkg/config"package redis

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
}*/

package redis

import (
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

func NewClient() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis:6379/0"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Failed to parse Redis URL:", err)
	}

	client := redis.NewClient(opt)
	log.Println("Connected to Redis at", redisURL)
	return client
}

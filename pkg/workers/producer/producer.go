package producer

import (
	"einvoice-access-point/pkg/config"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

type Producer struct {
	client *asynq.Client
}

func NewProducer() *Producer {
	redisUrl := fmt.Sprintf("%s:%s", config.Config.Redis.REDIS_HOST, config.Config.Redis.REDIS_PORT)
	redisConnection := asynq.RedisClientOpt{Addr: redisUrl, DB: 1}

	client := asynq.NewClient(redisConnection)
	return &Producer{client: client}
}

func (p *Producer) EnqueueTask(taskName string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	task := asynq.NewTask(taskName, payloadBytes)
	_, err = p.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %v", err)
	}
	defer p.client.Close()
	return nil
}

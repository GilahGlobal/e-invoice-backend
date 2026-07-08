package main

import (
	"einvoice-access-point/internal/config"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/database/postgresql"
	"einvoice-access-point/internal/utility"
	"einvoice-access-point/internal/workers/consumer"
	"einvoice-access-point/internal/workers/scheduler"
	"fmt"

	"github.com/hibiken/asynq"
)

func main() {
	logger := utility.NewLogger()
	if !logger.IsInitialized() {
		panic("Logger initialization failed: logger is nil")
	}
	configuration := config.Setup(logger, "./app")
	postgresql.ConnectToDatabase(logger, configuration.Database, configuration.TestDatabase)

	db, testDb := database.Connection()

	redisUrl := fmt.Sprintf("%s:%s", config.Config.Redis.REDIS_HOST, config.Config.Redis.REDIS_PORT)
	redisConnection := asynq.RedisClientOpt{Addr: redisUrl, DB: 1}

	scheduler := scheduler.NewScheduler(redisConnection)
	if err := scheduler.Start(); err != nil {
		logger.Error("Failed to start scheduler", "error", err)
	}

	consumer := consumer.NewQueueConsumer(db, testDb, redisConnection)
	if err := consumer.Start(); err != nil {
		logger.Error("Failed to start consumer", "error", err)
	}
}

package consumer

import (
	"einvoice-access-point/pkg/database"
	"einvoice-access-point/pkg/utility"
	bulkupload "einvoice-access-point/pkg/workers/consumer/bulk-upload"
	"log"

	"github.com/hibiken/asynq"
)

const TypeBulkUpload = "bulk:upload"

type QueueConsumer struct {
	svr    *asynq.Server
	logger *utility.Logger

	bulkupload *bulkupload.BulkUploadConsumer
}

func NewQueueConsumer(db, testDB *database.Database, redisConnection asynq.RedisClientOpt) *QueueConsumer {
	svr := asynq.NewServer(
		redisConnection,
		asynq.Config{
			Concurrency: 10,
		},
	)
	bulkupload := bulkupload.NewBulkUploadConsumer(db, testDB, utility.NewLogger())
	return &QueueConsumer{svr: svr, logger: utility.NewLogger(), bulkupload: bulkupload}
}

func (qc *QueueConsumer) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TypeBulkUpload, qc.bulkupload.HandleBulkUploadTask)
	log.Println("Asynq worker started, listening for jobs...")
	return qc.svr.Run(mux)
}

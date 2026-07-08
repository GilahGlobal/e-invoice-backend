package consumer

import (
	"einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/utility"
	bulkupload "einvoice-access-point/internal/workers/consumer/bulk-upload"
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
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBImpl := dbinit.InitDB(testDB.Postgresql.DB(), false)
	repo := repositories.NewInvoiceRepository(prodDB, testDBImpl)
	tokenRepo := repositories.NewTokenRepository(prodDB, testDBImpl)
	tokenSvc := token.NewService(tokenRepo)
	businessRepo := repositories.NewBusinessRepository(prodDB, testDBImpl)
	businessSvc := business.NewService(businessRepo)
	invoiceSvc := invoice.NewService(repo, tokenSvc, businessSvc)
	bulkRepo := repositories.NewBulkUploadRepository(prodDB, testDBImpl)
	bulkUploadSvc := bulk_upload.NewService(bulkRepo)
	subRepo := repositories.NewSubscriptionRepository(prodDB, testDBImpl)
	subSvc := subscription.NewService(subRepo, businessRepo)
	bulkupload := bulkupload.NewBulkUploadConsumer(db, testDB, utility.NewLogger(), invoiceSvc, bulkUploadSvc, subSvc)
	return &QueueConsumer{svr: svr, logger: utility.NewLogger(), bulkupload: bulkupload}
}

func (qc *QueueConsumer) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TypeBulkUpload, qc.bulkupload.HandleBulkUploadTask)
	log.Println("Asynq worker started, listening for jobs...")
	return qc.svr.Run(mux)
}

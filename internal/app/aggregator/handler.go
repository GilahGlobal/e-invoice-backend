package aggregator

import (
	"einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/utility"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc           *Service
	subSvc        *subscription.Service
	businessSvc   *business.Service
	invoiceSvc    *invoice.Service
	bulkUploadSvc *bulk_upload.Service
	Validator     *validator.Validate
	Logger        *utility.Logger
	Db            *database.Database
	TestDb        *database.Database
}

func NewHandler(validator *validator.Validate, logger *utility.Logger, db, testDb *database.Database) *Handler {
	prodDB := dbinit.InitDB(db.Postgresql.DB(), false)
	testDBConn := dbinit.InitDB(testDb.Postgresql.DB(), false)

	businessSvc := business.NewServiceWithDB(prodDB, testDBConn)
	bulkUploadSvc := bulk_upload.NewServiceWithDB(prodDB, testDBConn)
	tokenSvc := token.NewServiceWithDB(prodDB, testDBConn)
	invoiceSvc := invoice.NewServiceWithDB(prodDB, testDBConn, tokenSvc, businessSvc)
	subSvc := subscription.NewServiceWithDB(prodDB, testDBConn)

	svc := NewServiceWithDB(prodDB, testDBConn, subSvc, businessSvc, bulkUploadSvc)

	return &Handler{
		svc:           svc,
		subSvc:        subSvc,
		businessSvc:   businessSvc,
		invoiceSvc:    invoiceSvc,
		bulkUploadSvc: bulkUploadSvc,
		Validator:     validator,
		Logger:        logger,
		Db:            db,
		TestDb:        testDb,
	}
}

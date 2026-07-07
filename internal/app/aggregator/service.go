package aggregator

import (
	"einvoice-access-point/internal/app/bulk_upload"
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/subscription"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/repositories"
)

type Service struct {
	repo          repositories.AggregatorRepository
	subSvc        *subscription.Service
	businessSvc   *business.Service
	bulkUploadSvc *bulk_upload.Service
}

func NewService(repo repositories.AggregatorRepository, subSvc *subscription.Service, businessSvc *business.Service, bulkUploadSvc *bulk_upload.Service) *Service {
	return &Service{repo: repo, subSvc: subSvc, businessSvc: businessSvc, bulkUploadSvc: bulkUploadSvc}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager, subSvc *subscription.Service, businessSvc *business.Service, bulkUploadSvc *bulk_upload.Service) *Service {
	repo := repositories.NewAggregatorRepository(prodDb, testDb)
	return NewService(repo, subSvc, businessSvc, bulkUploadSvc)
}

package invoice

import (
	"einvoice-access-point/internal/app/business"
	"einvoice-access-point/internal/app/token"
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/repositories"
)

type Service struct {
	repo        repositories.InvoiceRepository
	tokenSvc    *token.Service
	businessSvc *business.Service
}

func NewService(repo repositories.InvoiceRepository, tokenSvc *token.Service, businessSvc *business.Service) *Service {
	return &Service{repo: repo, tokenSvc: tokenSvc, businessSvc: businessSvc}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager, tokenSvc *token.Service, businessSvc *business.Service) *Service {
	repo := repositories.NewInvoiceRepository(prodDb, testDb)
	return NewService(repo, tokenSvc, businessSvc)
}

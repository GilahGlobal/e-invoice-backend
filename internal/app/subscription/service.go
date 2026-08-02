package subscription

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/repositories"
)

type Service struct {
	repo         *repositories.SubscriptionRepository
	businessRepo *repositories.BusinessRepository
}

func NewService(repo *repositories.SubscriptionRepository, businessRepo *repositories.BusinessRepository) *Service {
	return &Service{repo: repo, businessRepo: businessRepo}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager) *Service {
	repo := repositories.NewSubscriptionRepository(prodDb, testDb)
	businessRepo := repositories.NewBusinessRepository(prodDb, testDb)
	return NewService(repo, businessRepo)
}

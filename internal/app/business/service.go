package business

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/repositories"
)

type Service struct {
	repo *repositories.BusinessRepository
}

func NewService(repo *repositories.BusinessRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func NewServiceWithDB(prodDB, testDB database.DatabaseManager) *Service {
	repo := repositories.NewBusinessRepository(prodDB, testDB)
	return NewService(repo)
}

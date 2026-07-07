package token

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/repositories"
)

type Service struct {
	repo repositories.TokenRepository
}

func NewService(repo repositories.TokenRepository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager) *Service {
	repo := repositories.NewTokenRepository(prodDb, testDb)
	return NewService(repo)
}

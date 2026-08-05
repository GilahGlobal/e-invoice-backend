package token

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/dbinit"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/data/repositories"
	"einvoice-access-point/internal/pkg/zoho"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo *repositories.TokenRepository
}

func NewService(repo *repositories.TokenRepository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithDB(prodDb, testDb database.DatabaseManager) *Service {
	repo := repositories.NewTokenRepository(prodDb, testDb)
	return NewService(repo)
}

func (s *Service) GetValidAccessToken(db *gorm.DB, accConfig entities.AccountingPlatformConfig, provider, orgID string, code ...string) (string, error) {
	pdb := dbinit.InitDB(db, false)

	token, err := s.repo.FindToken(pdb, provider, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if len(code) == 0 {
				return "", errors.New("authorization code required for new token")
			}
			newToken, err := zoho.ExchangeCodeForTokens(code[0], string(accConfig.APIKey), string(accConfig.APISecret))
			if err != nil {
				return "", err
			}

			if err := s.repo.SaveNewZohoToken(pdb, provider, orgID, newToken.AccessToken, newToken.RefreshToken, newToken.ExpiresIn); err != nil {
				return "", err
			}

			return newToken.AccessToken, nil
		}
		return "", err
	}

	if time.Now().After(token.ExpiresAt.Add(-5 * time.Minute)) {
		newToken, err := zoho.RefreshAccessToken(token.RefreshToken, string(accConfig.APIKey), string(accConfig.APISecret))
		if err != nil {
			log.Println("zoho error ", err)
			return "", err
		}

		refreshToken := token.RefreshToken
		if newToken.RefreshToken != "" {
			refreshToken = newToken.RefreshToken
		}

		if err := s.repo.UpdateZohoToken(pdb, provider, orgID, newToken.AccessToken, refreshToken, newToken.ExpiresIn); err != nil {
			return "", err
		}
		return newToken.AccessToken, nil
	}

	return token.AccessToken, nil
}

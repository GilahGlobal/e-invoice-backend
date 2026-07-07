package repositories

import (
	"time"

	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
)

type TokenRepository interface {
	FindToken(db database.DatabaseManager, provider, orgID string) (*entities.TokenManager, error)
	SaveNewZohoToken(db database.DatabaseManager, provider, orgID, accessToken, refreshToken string, expiresIn int) error
	UpdateZohoToken(db database.DatabaseManager, provider, orgID, accessToken, refreshToken string, expiresIn int) error
}

type tokenRepository struct {
	prodDb database.DatabaseManager
	testDb database.DatabaseManager
}

func NewTokenRepository(prodDb, testDb database.DatabaseManager) TokenRepository {
	return &tokenRepository{prodDb: prodDb, testDb: testDb}
}

func (r *tokenRepository) FindToken(db database.DatabaseManager, provider, orgID string) (*entities.TokenManager, error) {
	var token entities.TokenManager
	if err := db.DB().Where("provider = ? AND organization_id = ?", provider, orgID).
		First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) SaveNewZohoToken(db database.DatabaseManager, provider, orgID, accessToken, refreshToken string, expiresIn int) error {
	zohoToken := entities.TokenManager{
		Provider:       provider,
		OrganizationID: orgID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresAt:      time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	return db.DB().Create(&zohoToken).Error
}

func (r *tokenRepository) UpdateZohoToken(db database.DatabaseManager, provider, orgID, accessToken, refreshToken string, expiresIn int) error {
	var zohoToken entities.TokenManager
	if err := db.DB().Where("provider = ? AND organization_id = ?", provider, orgID).First(&zohoToken).Error; err != nil {
		return err
	}

	zohoToken.AccessToken = accessToken
	zohoToken.RefreshToken = refreshToken
	zohoToken.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return db.DB().Save(&zohoToken).Error
}

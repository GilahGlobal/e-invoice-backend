package token

import (
	"einvoice-access-point/external/zoho"
	repository "einvoice-access-point/internal/repository/token"
	inst "einvoice-access-point/pkg/dbinit"
	"einvoice-access-point/pkg/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

func GetValidAccessToken(db *gorm.DB, accConfig models.AccountingPlatformConfig, provider, orgID string, code ...string) (string, error) {
	pdb := inst.InitDB(db, false)

	token, err := repository.FindToken(pdb, provider, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if len(code) == 0 {
				return "", errors.New("authorization code required for new token")
			}
			newToken, err := zoho.ExchangeCodeForTokens(code[0], string(accConfig.APIKey), string(accConfig.APISecret))
			if err != nil {
				return "", err
			}

			if err := repository.SaveNewZohoToken(pdb, provider, orgID, newToken.AccessToken, newToken.RefreshToken, newToken.ExpiresIn); err != nil {
				return "", err
			}

			return newToken.AccessToken, nil
		}
		return "", err
	}

	if time.Now().After(token.ExpiresAt.Add(-5 * time.Minute)) {
		newToken, err := zoho.RefreshAccessToken(token.RefreshToken, string(accConfig.APIKey), string(accConfig.APISecret))
		if err != nil {
			return "", err
		}

		refreshToken := token.RefreshToken
		if newToken.RefreshToken != "" {
			refreshToken = newToken.RefreshToken
		}

		if err := repository.UpdateZohoToken(pdb, provider, orgID, newToken.AccessToken, refreshToken, newToken.ExpiresIn); err != nil {
			return "", err
		}
		return newToken.AccessToken, nil
	}

	return token.AccessToken, nil
}

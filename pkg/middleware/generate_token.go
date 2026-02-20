package middleware

import (
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func CreateToken(user models.Business, isSandbox bool) (*TokenDetailDTO, error) {
	var (
		configs   = config.GetConfig()
		tokenData = &TokenDetailDTO{}
		err       error
	)

	tokenData.AccessUuid = utility.GenerateUUID()
	expireDuration := configs.Server.AccessTokenExpireDuration
	tokenData.ExpiresAt = time.Now().Add(time.Duration(expireDuration) * time.Hour)

	theClaims := UserDataClaims{
		user.ID,
		user.Name,
		user.Email,
		user.BusinessID,
		user.ServiceID,
		tokenData.AccessUuid,
		isSandbox,
		jwt.RegisteredClaims{
			Issuer:    user.Email,
			ExpiresAt: jwt.NewNumericDate(tokenData.ExpiresAt),
		},
	}
	claims := jwt.NewWithClaims(jwt.SigningMethodHS512, theClaims)
	tokenData.AccessToken, err = claims.SignedString([]byte(configs.Server.Secret))
	if err != nil {
		return tokenData, err
	}

	return tokenData, err

}

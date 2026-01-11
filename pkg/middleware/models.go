package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Authorization struct {
	Authorization string `reqHeader:"Authorization"`
}

type UserDataClaims struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	BusinessID *string `json:"business_id"`
	ServiceID  string  `json:"service_id"`
	AccessUuid string  `json:"access_uuid"`
	jwt.RegisteredClaims
}

type TokenDetailDTO struct {
	AccessUuid  string `json:"access_uuid"`
	AccessToken string `json:"access_token"`
	ExpiresAt   time.Time
}

package admin

import "einvoice-access-point/internal/data/entities"

type AdminLoginRequestDto struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AdminRegisterDto struct {
	Name     string             `json:"name" validate:"required"`
	Email    string             `json:"email" validate:"required,email"`
	Password string             `json:"password" validate:"required,min=8"`
	Role     entities.AdminRole `json:"role" validate:"required,oneof=superadmin admin"`
}

type AdminResponse struct {
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Email string             `json:"email"`
	Role  entities.AdminRole `json:"role"`
}

type AdminLoginResponseDto struct {
	Data        AdminResponse `json:"data"`
	AccessToken string        `json:"access_token"`
}

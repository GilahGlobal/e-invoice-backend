package dtos

type BaseResponseDto struct {
	Status     string `json:"status" example:"success"`
	StatusCode int    `json:"status_code" example:"200"`
	Message    string `json:"message" example:"Action performed successfully"`
}

type RegisterDto struct {
	Name            string              `json:"name" example:"John Doe" validate:"required,min=2,max=250"`
	Email           string              `json:"email" example:"john.doe@example.com" validate:"required,email"`
	Password        string              `json:"password" example:"password123" validate:"required,min=6"`
	CompanyName     string              `json:"company_name" example:"Acme Inc." validate:"required"`
	TIN             string              `json:"tin" example:"TIN-123456789" validate:"required"`
	PhoneNumber     string              `json:"phone_number" example:"+1234567890" validate:"required"`
	PlatformConfigs PlatformConfigsAuth `json:"platform_configs" validate:"dive"`
}

type RegisterResponseDto struct {
	BaseResponseDto
	Data struct {
		ID         string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
		Name       string  `json:"name" example:"John Doe"`
		Email      string  `json:"email" example:"john.doe@example.com"`
		BusinessID *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
		ServiceID  string  `json:"service_id" example:"123e4567-e89b-12d3-a456-426614174000"`
		Tin        string  `json:"tin" example:"TIN-123456789"`
	} `json:"data"`
}

type UpdateUserRequestModel struct {
	Name string `json:"name" validate:"required"`
}

type LoginRequestDto struct {
	Email     string `json:"email" validate:"required"`
	Password  string `json:"password" validate:"required"`
	IsSandbox bool   `json:"is_sandbox" default:"true" validate:"omitempty"`
}
type UserResponse struct {
	ID         string  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email      string  `json:"email" example:"john.doe@example.com"`
	Name       string  `json:"name" example:"John Doe"`
	BusinessID *string `json:"business_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	ServiceID  string  `json:"service_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	IsSandbox  bool    `json:"is_sandbox" example:"true"`
}
type LoginResponseDto struct {
	BaseResponseDto
	Data        UserResponse `json:"data"`
	AccessToken string       `json:"access_token"`
}

type InitiateForgotPasswordDto struct {
	Email string `json:"email" example:"john.doe@example.com" validate:"required,email"`
}

type CompleteForgotPasswordDto struct {
	Email    string `json:"email" example:"john.doe@example.com" validate:"required,email"`
	OTP      string `json:"otp" example:"123456" validate:"required"`
	Password string `json:"password" example:"password123" validate:"required,min=6"`
}

type PlatformConfigsAuth map[string]AccountingPlatformConfigAuth
type AccountingPlatformConfigAuth struct {
	OrgID      string `json:"org_id" example:"org-123456789"`
	AuthToken  string `json:"auth_token" example:"auth-token-123456789"`
	HMACSecret string `json:"hmac_secret" example:"hmac-secret-123456789"`
	APIKey     string `json:"api_key" example:"api-key-123456789"`
	APISecret  string `json:"api_secret" example:"api-secret-123456789"`
}

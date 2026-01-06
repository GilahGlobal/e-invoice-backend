package models

type CreateUserRequestModel struct {
	Name            string              `json:"name" validate:"required,min=2,max=250"`
	Email           string              `json:"email" validate:"required,email"`
	Password        string              `json:"password" validate:"required,min=6"`
	CompanyName     string              `json:"company_name" validate:"required"`
	TIN             string              `json:"tin" validate:"required"`
	PhoneNumber     string              `json:"phone_number" validate:"required"`
	PlatformConfigs PlatformConfigsAuth `json:"platform_configs" validate:"dive"`
}

type UpdateUserRequestModel struct {
	Name string `json:"name" validate:"required"`
}

type LoginRequestModel struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type InitiateForgotPassword struct {
	Email string `json:"email" validate:"required,email"`
}

type CompleteForgotPassword struct {
	Email    string `json:"email" validate:"required,email"`
	OTP      string `json:"otp" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	BusinessID string `json:"business_id"`
	ServiceID  string `json:"service_id"`
	Tin        string `json:"tin"`
}

type PlatformConfigsAuth map[string]AccountingPlatformConfigAuth
type AccountingPlatformConfigAuth struct {
	OrgID      string `json:"org_id"`
	AuthToken  string `json:"auth_token"`
	HMACSecret string `json:"hmac_secret"`
	APIKey     string `json:"api_key"`
	APISecret  string `json:"api_secret"`
}

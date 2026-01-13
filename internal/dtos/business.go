package dtos

type UpdateBusinessDto struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=250"`
	Email       *string `json:"email" validate:"omitempty,email"`
	PhoneNumber *string `json:"phone_number" validate:"omitempty,numeric"`
	CompanyName *string `json:"company_name" validate:"omitempty,max=250"`
}

type UpdateBusinessIDRequest struct {
	BusinessID string `json:"business_id" validate:"required,uuid"`
}

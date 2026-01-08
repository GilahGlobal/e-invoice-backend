package models

type UpdateBusinessIDRequest struct {
	BusinessID string `json:"business_id" validate:"required,uuid"`
}

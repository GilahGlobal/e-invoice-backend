package invoice

import (
	"einvoice-access-point/internal/data/database"
	"einvoice-access-point/internal/data/entities"
	"einvoice-access-point/internal/pkg/firs_models"
)

type InvoiceStepMetadata struct {
	Step      string `json:"step" example:"validated_irn"`
	Status    string `json:"status" example:"success"`
	Timestamp string `json:"timestamp" example:"2024-01-01T12:00:00Z"`
}

type InvoiceData struct {
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	QRCode2       string `json:"qr_code_2" example:"eeleGz7LXrt3gignmXGi9DAeXoVS7GjMR/8WK4f8G76DSP14SA2PSyArr4oaS6ojo0EqCTlp2UBjT2eRpn51..."`
	QRCodeBMP     string `json:"qr_code_bmp" example:"Qk02AAAAAAAAAAAAAAAoAAAAAQAAAAEAAAABAAEAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAP..."`
}

type InvoiceUploadData struct {
	ID            string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber string `json:"invoice_number" example:"INV-1001"`
	IRN           string `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	QRCode        string `json:"qr_code" example:"iVBORw0KGgoAAAANSUhEUgAAAQAAAAEAAQMAAABmvDolAAAABlBMVEX///8AAABVwtN..."`
	QRCodeBMPURL  string `json:"qr_code_bmp_url" example:"https://res.cloudinary.com/demo/image/upload/v1712345678/invoice-bmp-123.bmp"`
}

type UploadInvoiceResponseDto struct {
	entities.Response
	Data     InvoiceUploadData     `json:"data"`
	Metadata []InvoiceStepMetadata `json:"metadata"`
}

type GetAllInvoicesResponseDto struct {
	entities.Response
	Data       []entities.MinimalInvoiceDTO `json:"data"`
	Pagination database.PaginationResponse  `json:"pagination"`
}

type ResourceItemDto struct {
	Code  string `json:"code" example:"380"`
	Value string `json:"value" example:"Credit Note"`
}

type ResourcesResponseDto struct {
	entities.Response
	Data []ResourceItemDto `json:"data"`
}

type HSNCodesItemDto struct {
	HSCode      string `json:"hscode" example:"0101.21"`
	Description string `json:"description" example:"Horses; live, pure-bred breeding animals"`
}

type HSNCodesResponseDto struct {
	entities.Response
	Data []HSNCodesItemDto `json:"data"`
}

type ServiceCodesItemDto struct {
	Code        string `json:"code" example:"0111"`
	Description string `json:"description" example:"Growing of cereals (except rice), leguminous crops and oil seeds"`
}

type ServiceCodesResponseDto struct {
	entities.Response
	Data []ServiceCodesItemDto `json:"data"`
}

type TaxCategoryItemDto struct {
	Code    string `json:"code" example:"STANDARD_GST"`
	Value   string `json:"value" example:"Standard Goods and Services Tax"`
	Percent string `json:"percent" example:"Not Available"`
}

type TaxCategoriesResponseDto struct {
	entities.Response
	Data []TaxCategoryItemDto `json:"data"`
}

type CountryItemDto struct {
	Name                   string `json:"name" example:"Afghanistan"`
	Alpha2                 string `json:"alpha_2" example:"AF"`
	Alpha3                 string `json:"alpha_3" example:"AFG"`
	CountryCode            string `json:"country_code" example:"004"`
	Iso31662               string `json:"iso_3166_2" example:"ISO 3166-2:AF"`
	Region                 string `json:"region" example:"Asia"`
	SubRegion              string `json:"sub_region" example:"Southern Asia"`
	IntermediateRegion     string `json:"intermediate_region" example:""`
	RegionCode             string `json:"region_code" example:"142"`
	SubRegionCode          string `json:"sub_region_code" example:"034"`
	IntermediateRegionCode string `json:"intermediate_region_code" example:""`
}

type CountriesResponseDto struct {
	entities.Response
	Data []CountryItemDto `json:"data"`
}

type CurrencyItemDto struct {
	Symbol        string `json:"symbol" example:"$"`
	Name          string `json:"name" example:"US Dollar"`
	SymbolNative  string `json:"symbol_native" example:"$"`
	DecimalDigits int    `json:"decimal_digits" example:"2"`
	Rounding      int    `json:"rounding" example:"0"`
	Code          string `json:"code" example:"USD"`
	NamePlural    string `json:"name_plural" example:"US dollars"`
}

type CurrenciesResponseDto struct {
	entities.Response
	Data []CurrencyItemDto `json:"data"`
}

type LGAItemDto struct {
	Name      string `json:"name" example:"Aba North"`
	Code      string `json:"code" example:"NG-AB-ANO"`
	StateCode string `json:"state_code" example:"NG-AB"`
}

type LGAsResponseDto struct {
	entities.Response
	Data []LGAItemDto `json:"data"`
}

type StateItemDto struct {
	Name string `json:"name" example:"Abia"`
	Code string `json:"code" example:"NG-AB"`
}

type StatesResponseDto struct {
	entities.Response
	Data []StateItemDto `json:"data"`
}

type GetInvoiceDetailsResponseDto struct {
	entities.Response
	Data entities.Invoice `json:"data"`
}

type GetInvoiceStatsResponseDto struct {
	entities.Response
	Data entities.InvoiceStatsResponseData `json:"data"`
}

type InvoiceResponse struct {
	ID               string                              `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	InvoiceNumber    string                              `json:"invoice_number" example:"INV-1001"`
	IRN              string                              `json:"irn" example:"123e4567-e89b-12d3-a456-426614174000"`
	BusinessID       string                              `json:"business_id" example:"business-uuid"`
	Platform         string                              `json:"platform" example:"zoho"`
	PlatformMetadata string                              `json:"platform_metadata"`
	InvoiceData      firs_models.UploadInvoiceRequestDto `json:"invoice_data"`
	CurrentStatus    string                              `json:"current_status" example:"validated_irn"`
	StatusHistory    []InvoiceStepMetadata               `json:"status_history"`
	Timestamp        string                              `json:"timestamp" example:"2024-01-01T12:00:00Z"`
	CreatedAt        string                              `json:"created_at" example:"2024-01-01T12:00:00Z"`
	UpdatedAt        string                              `json:"updated_at" example:"2024-01-02T12:00:00Z"`
}

type GetInvoiceResponseDto struct {
	entities.Response
	Data InvoiceResponse `json:"data"`
}

type BulkUpdateInvoiceResponseDto struct {
	entities.Response
	Data firs_models.BulkUpdateInvoiceResponse `json:"data"`
}

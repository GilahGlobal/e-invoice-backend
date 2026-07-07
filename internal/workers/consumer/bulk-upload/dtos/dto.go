package dtos

import (
	invoicePkg "einvoice-access-point/internal/app/invoice"
	"einvoice-access-point/internal/pkg/firs_models"
)

type UploadInvoiceRequestDto = firs_models.UploadInvoiceRequestDto
type InvoiceDeliveryPeriod = firs_models.InvoiceDeliveryPeriod
type DocumentReference = firs_models.DocumentReference
type Party = firs_models.Party
type PostalAddress = firs_models.PostalAddress
type PaymentMeans = firs_models.PaymentMeans
type AllowanceCharge = firs_models.AllowanceCharge
type TaxTotal = firs_models.TaxTotal
type TaxSubtotal = firs_models.TaxSubtotal
type TaxCategory = firs_models.TaxCategory
type LegalMonetaryTotal = firs_models.LegalMonetaryTotal
type InvoiceLine = firs_models.InvoiceLine
type Item = firs_models.Item
type Price = firs_models.Price

type InvoiceData = invoicePkg.InvoiceData

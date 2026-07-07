package zohoinvoice

import (
	"einvoice-access-point/internal/pkg/firs_models"
	"einvoice-access-point/internal/pkg/zoho"
	"einvoice-access-point/internal/utility"
	"fmt"
	"time"
)

// ConvertZohoToFIRS converts a Zoho invoice to FIRS invoice format.
func ConvertZohoToFIRS(zohoInvoice zoho.Invoice, organizationID, organizationName, irn string) (firs_models.UploadInvoiceRequestDto, error) {
	createdTime, err := time.Parse("2006-01-02T15:04:05-0700", zohoInvoice.CreatedTime)
	if err != nil {
		return firs_models.UploadInvoiceRequestDto{}, fmt.Errorf("failed to parse created_time: %v", err)
	}

	issueTime := createdTime.Format("15:04:05")
	contactPhone := utility.FormatPhone(zohoInvoice.ContactPersonsDetails[0].Phone)
	zohoStatus := mapStatus(zohoInvoice.Status)

	var taxTotal float64
	var taxSubtotals []firs_models.TaxSubtotal
	for _, item := range zohoInvoice.LineItems {
		if item.TaxPercentage > 0 {
			taxAmount := item.ItemTotal * (item.TaxPercentage / 100)
			taxTotal += taxAmount
			taxSubtotals = append(taxSubtotals, firs_models.TaxSubtotal{
				TaxableAmount: item.ItemTotal,
				TaxAmount:     taxAmount,
				TaxCategory: firs_models.TaxCategory{
					ID:      *item.TaxID,
					Percent: item.TaxPercentage,
				},
			})
		}
	}

	firsInvoice := firs_models.UploadInvoiceRequestDto{
		BusinessID:           organizationID,
		IRN:                  &irn,
		IssueDate:            zohoInvoice.Date,
		DueDate:              &zohoInvoice.DueDate,
		IssueTime:            &issueTime,
		InvoiceTypeCode:      "381",
		PaymentStatus:        &zohoStatus,
		Note:                 &zohoInvoice.Notes,
		TaxPointDate:         &zohoInvoice.Date,
		DocumentCurrencyCode: zohoInvoice.CurrencyCode,
		TaxCurrencyCode:      zohoInvoice.CurrencyCode,
		AccountingSupplierParty: firs_models.Party{
			PartyName: organizationName,
			TIN:       "TIN-UNKNOWN",
			Email:     "supplier@example.com",
			PostalAddress: firs_models.PostalAddress{
				StreetName: "test adress",
				CityName:   "amac",
				PostalZone: "19001",
				Country:    "NG",
				LGA:        "NG-LA-LIS",
				State:      "NG-LA",
			},
		},
		AccountingCustomerParty: &firs_models.Party{
			PartyName: zohoInvoice.CustomerName,
			TIN:       "TIN-" + zohoInvoice.CustomerID,
			Email:     zohoInvoice.Email,
			Telephone: &contactPhone,
			PostalAddress: firs_models.PostalAddress{
				StreetName: zohoInvoice.BillingAddress.Address,
				CityName:   zohoInvoice.BillingAddress.City,
				PostalZone: zohoInvoice.BillingAddress.Zip,
				Country:    zohoInvoice.BillingAddress.CountryCode,
				LGA:        "NG-LA-LIS",
				State:      "NG-LA",
			},
		},
		PaymentTermsNote: &zohoInvoice.Terms,
		TaxTotal: []firs_models.TaxTotal{
			{
				TaxAmount:   taxTotal,
				TaxSubtotal: taxSubtotals,
			},
		},
		LegalMonetaryTotal: firs_models.LegalMonetaryTotal{
			LineExtensionAmount: zohoInvoice.Total - taxTotal,
			TaxExclusiveAmount:  zohoInvoice.Total - taxTotal,
			TaxInclusiveAmount:  zohoInvoice.Total,
			PayableAmount:       zohoInvoice.Total,
		},
	}

	for _, item := range zohoInvoice.LineItems {
		firsInvoice.InvoiceLine = append(firsInvoice.InvoiceLine, firs_models.InvoiceLine{
			HSNCode:             item.ItemID,
			ProductCategory:     "General",
			InvoicedQuantity:    int(item.Quantity),
			LineExtensionAmount: item.ItemTotal,
			Item: firs_models.Item{
				Name:        item.Name,
				Description: item.Description,
			},
			Price: firs_models.Price{
				PriceAmount:  item.Rate,
				BaseQuantity: int(item.Quantity),
				PriceUnit:    zohoInvoice.CurrencyCode + " per 1",
			},
		})
	}

	return firsInvoice, nil
}

func mapStatus(zohoStatus string) string {
	switch zohoStatus {
	case "paid":
		return "PAID"
	case "sent", "draft":
		return "PENDING"
	default:
		return "PENDING"
	}
}

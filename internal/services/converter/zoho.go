package converter

import (
	"einvoice-access-point/external/zoho"
	"einvoice-access-point/internal/dtos"
	"einvoice-access-point/pkg/utility"
	"fmt"
	"time"
)

// ConvertZohoToFIRS converts a Zoho invoice to FIRS invoice format
func ConvertZohoToFIRS(zoho zoho.Invoice, organizationID, organizationName, irn string) (dtos.UploadInvoiceRequestDto, error) {
	// Parse created time to extract issue time
	createdTime, err := time.Parse("2006-01-02T15:04:05-0700", zoho.CreatedTime)
	if err != nil {
		return dtos.UploadInvoiceRequestDto{}, fmt.Errorf("failed to parse created_time: %v", err)
	}
	issueTime := createdTime.Format("15:04:05")
	contactPhone := utility.FormatPhone(zoho.ContactPersonsDetails[0].Phone)
	zohotatus := mapStatus(zoho.Status)

	// Calculate tax totals
	var taxTotal float64
	var taxSubtotals []dtos.TaxSubtotal
	for _, item := range zoho.LineItems {
		if item.TaxPercentage > 0 {
			taxAmount := item.ItemTotal * (item.TaxPercentage / 100)
			taxTotal += taxAmount
			taxSubtotals = append(taxSubtotals, dtos.TaxSubtotal{
				TaxableAmount: item.ItemTotal,
				TaxAmount:     taxAmount,
				TaxCategory: dtos.TaxCategory{
					ID:      *item.TaxID,
					Percent: item.TaxPercentage,
				},
			})
		}
	}

	// Map Zoho invoice to FIRS invoice
	firsInvoice := dtos.UploadInvoiceRequestDto{
		BusinessID:           organizationID, // Using organization_id as business_id
		IRN:                  &irn,
		IssueDate:            zoho.Date,
		DueDate:              &zoho.DueDate,
		IssueTime:            &issueTime,
		InvoiceTypeCode:      "381", // Default invoice type code
		PaymentStatus:        &zohotatus,
		Note:                 &zoho.Notes,
		TaxPointDate:         &zoho.Date, // Using invoice_date as tax_point_date
		DocumentCurrencyCode: zoho.CurrencyCode,
		TaxCurrencyCode:      zoho.CurrencyCode,
		AccountingSupplierParty: dtos.Party{
			PartyName: organizationName,
			TIN:       "TIN-UNKNOWN",          // Placeholder, as TIN is not provided in Zoho data
			Email:     "supplier@example.com", // Placeholder, as supplier email is not provided                  // Optional, not provided in Zoho data
			PostalAddress: dtos.PostalAddress{
				StreetName: "test adress", // Not provided in Zoho data
				CityName:   "amac",
				PostalZone: "19001",
				Country:    "NG",
				LGA:        "NG-LA-LIS",
				State:      "NG-LA",
			},
		},
		AccountingCustomerParty: &dtos.Party{
			PartyName: zoho.CustomerName,
			TIN:       "TIN-" + zoho.CustomerID, // Using customer_id as part of TIN
			Email:     zoho.Email,
			Telephone: &contactPhone,
			PostalAddress: dtos.PostalAddress{
				StreetName: zoho.BillingAddress.Address,
				CityName:   zoho.BillingAddress.City,
				PostalZone: zoho.BillingAddress.Zip,
				Country:    zoho.BillingAddress.CountryCode,
				LGA:        "NG-LA-LIS",
				State:      "NG-LA",
			},
		},
		PaymentTermsNote: &zoho.Terms,
		TaxTotal: []dtos.TaxTotal{
			{
				TaxAmount:   taxTotal,
				TaxSubtotal: taxSubtotals,
			},
		},
		LegalMonetaryTotal: dtos.LegalMonetaryTotal{
			LineExtensionAmount: zoho.Total - taxTotal,
			TaxExclusiveAmount:  zoho.Total - taxTotal,
			TaxInclusiveAmount:  zoho.Total,
			PayableAmount:       zoho.Total,
		},
	}

	// Map line items
	for _, item := range zoho.LineItems {
		firsInvoice.InvoiceLine = append(firsInvoice.InvoiceLine, dtos.InvoiceLine{
			HSNCode:             item.ItemID, // Using item_id as HSN code
			ProductCategory:     "General",   // Placeholder, as category is not provided
			InvoicedQuantity:    int(item.Quantity),
			LineExtensionAmount: item.ItemTotal,
			Item: dtos.Item{
				Name:        item.Name,
				Description: item.Description,
			},
			Price: dtos.Price{
				PriceAmount:  item.Rate,
				BaseQuantity: int(item.Quantity),
				PriceUnit:    zoho.CurrencyCode + " per 1",
			},
		})
	}

	return firsInvoice, nil
}

// mapStatus maps Zoho invoice status to FIRS payment status
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

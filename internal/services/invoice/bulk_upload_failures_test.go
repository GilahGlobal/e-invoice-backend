package invoice

import (
	"encoding/json"
	"testing"
)

func TestParseBulkUploadFailedInvoices(t *testing.T) {
	t.Run("returns empty slice for empty payload", func(t *testing.T) {
		failures, err := parseBulkUploadFailedInvoices(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(failures) != 0 {
			t.Fatalf("expected no failures, got %d", len(failures))
		}
	})

	t.Run("parses invoice payload alongside errors", func(t *testing.T) {
		raw := json.RawMessage(`[
			{
				"invoice_index":1,
				"invoice_number":"INV-001",
				"invoice":{
					"invoice_number":"INV-001",
					"business_id":"123e4567-e89b-12d3-a456-426614174000",
					"issue_date":"2026-01-16",
					"invoice_type_code":"380",
					"document_currency_code":"NGN",
					"tax_currency_code":"NGN",
					"accounting_supplier_party":{
						"party_name":"Acme Inc.",
						"tin":"123456789012345",
						"email":"supplier@example.com",
						"postal_address":{
							"street_name":"123 Broad Street",
							"city_name":"Ikeja",
							"postal_zone":"100001",
							"lga":"NG-LA-IKJ",
							"state":"NG-LA",
							"country":"NG"
						}
					},
					"tax_total":[],
					"legal_monetary_total":{
						"line_extension_amount":100,
						"tax_exclusive_amount":100,
						"tax_inclusive_amount":100,
						"payable_amount":100
					},
					"invoice_line":[]
				},
				"error":"duplicate invoice sent"
			},
			{
				"invoice_index":2,
				"invoice_number":"INV-002",
				"error":{"details":"invalid tax amount"}
			}
		]`)

		failures, err := parseBulkUploadFailedInvoices(raw)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(failures) != 2 {
			t.Fatalf("expected 2 failures, got %d", len(failures))
		}
		if failures[0].Invoice == nil || failures[0].Invoice.InvoiceNumber != "INV-001" {
			t.Fatalf("expected first failure to include invoice payload, got %#v", failures[0].Invoice)
		}

		objectError, ok := failures[1].Error.(map[string]any)
		if !ok {
			t.Fatalf("expected second error to be an object, got %T", failures[1].Error)
		}
		if objectError["details"] != "invalid tax amount" {
			t.Fatalf("unexpected object error payload: %#v", objectError)
		}
	})

	t.Run("returns error for invalid payload", func(t *testing.T) {
		_, err := parseBulkUploadFailedInvoices(json.RawMessage(`{`))
		if err == nil {
			t.Fatal("expected an error for invalid json payload")
		}
	})
}

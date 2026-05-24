package invoice

import (
	"einvoice-access-point/internal/dtos"
	"encoding/json"
	"strings"
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
				"stage":"validated_invoice",
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
		if failures[0].Stage != "validated_invoice" {
			t.Fatalf("expected first failure stage to be parsed, got %q", failures[0].Stage)
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

func TestNormalizeBulkUploadFailureReason(t *testing.T) {
	reason := normalizeBulkUploadFailureReason(map[string]any{"detail": "invalid tax amount"})
	if !strings.Contains(reason, "invalid tax amount") {
		t.Fatalf("expected normalized reason to contain serialized object, got %q", reason)
	}
}

func TestBuildBulkUploadFailedInvoiceExportRows(t *testing.T) {
	rows := BuildBulkUploadFailedInvoiceExportRows(&dtos.BulkUploadFailedInvoicesDto{
		FailedInvoices: []dtos.BulkUploadFailedInvoiceDto{
			{
				InvoiceNumber: "INV-001",
				Stage:         "validation",
				Reason:        "missing invoice line",
			},
		},
	})

	if len(rows) != 1 {
		t.Fatalf("expected 1 export row, got %d", len(rows))
	}
	if rows[0].Stage != "validation" || rows[0].Reason != "missing invoice line" {
		t.Fatalf("unexpected export row: %#v", rows[0])
	}
}

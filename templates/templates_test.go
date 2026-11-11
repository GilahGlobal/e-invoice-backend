package templates

import (
	"strings"
	"testing"
)

func TestTemplatesRendering(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		contains string
	}{
		{
			name:     "otp",
			template: "otp.html",
			data: map[string]any{
				"OTP": "123456",
			},
			contains: "123456",
		},
		{
			name:     "admin_created",
			template: "admin_created.html",
			data: map[string]any{
				"Email":    "admin@example.com",
				"Password": "tempPassword123",
			},
			contains: "admin@example.com",
		},
		{
			name:     "aggregator_invitation",
			template: "aggregator_invitation.html",
			data: map[string]any{
				"BusinessName": "Acme Corp",
			},
			contains: "Acme Corp",
		},
		{
			name:     "new_aggregator_invitation",
			template: "new_aggregator_invitation.html",
			data: map[string]any{
				"BusinessName": "Acme Corp",
				"SignupLink":   "https://example.com/signup?token=abc",
			},
			contains: "https://example.com/signup?token=abc",
		},
		{
			name:     "invitation_accepted",
			template: "invitation_accepted.html",
			data: map[string]any{
				"AggregatorName": "FastTrack Aggregator",
			},
			contains: "FastTrack Aggregator",
		},
		{
			name:     "invitation_rejected",
			template: "invitation_rejected.html",
			data: map[string]any{
				"AggregatorName": "FastTrack Aggregator",
			},
			contains: "FastTrack Aggregator",
		},
		{
			name:     "invoice_validation_completed",
			template: "invoice_validation_completed.html",
			data: map[string]any{
				"BusinessName":   "Acme Corp",
				"InvoiceNumber":  "INV-001",
				"AmountDisplay":  "NGN 50,000.00",
				"IRN":            "IRN-123-456",
				"ValidationTime": "Nov 10, 2026 10:00:00 UTC",
			},
			contains: "INV-001",
		},
		{
			name:     "invoice_validation_failed",
			template: "invoice_validation_failed.html",
			data: map[string]any{
				"BusinessName":  "Acme Corp",
				"InvoiceNumber": "INV-002",
				"FailureReason": "Tax ID is invalid",
				"Timestamp":     "Nov 10, 2026 10:00:00 UTC",
			},
			contains: "Tax ID is invalid",
		},
		{
			name:     "bulk_upload_finished",
			template: "bulk_upload_finished.html",
			data: map[string]any{
				"BusinessName":     "Acme Corp",
				"FileName":         "invoices_nov.xlsx",
				"TotalRows":        100,
				"Successful":       95,
				"Partial":          0,
				"Failed":           5,
				"ValidRows":        95,
				"ValidationErrors": 5,
				"Duration":         "2.4s",
			},
			contains: "invoices_nov.xlsx",
		},
		{
			name:     "subscription_expiring_reminder_urgent",
			template: "subscription_expiring_reminder.html",
			data: map[string]any{
				"RecipientName":    "Acme Corp",
				"PlanName":         "Enterprise Plan",
				"DaysRemaining":    7,
				"DaysText":         "7 days",
				"ExpirationDate":   "17 November 2026",
				"BillingCycleText": "Yearly",
				"ReferenceID":      "NXR-2026-004821",
				"AmountDue":        "₦120,000.00",
				"RemainingInvoices": 250,
				"RenewalURL":       "https://app.nexar.ng/dashboard/subscriptions",
			},
			contains: "DAYS LEFT TO RENEW",
		},
		{
			name:     "subscription_expiring_reminder_advance",
			template: "subscription_expiring_reminder.html",
			data: map[string]any{
				"RecipientName":    "Acme Corp",
				"PlanName":         "Enterprise Plan",
				"DaysRemaining":    30,
				"DaysText":         "30 days",
				"ExpirationDate":   "10 December 2026",
				"BillingCycleText": "Yearly",
				"ReferenceID":      "NXR-2026-004821",
				"AmountDue":        "₦120,000.00",
				"RemainingInvoices": 500,
				"RenewalURL":       "https://app.nexar.ng/dashboard/subscriptions",
			},
			contains: "PAYMENT SUMMARY",
		},
		{
			name:     "subscription_payment_receipt",
			template: "subscription_payment_receipt.html",
			data: map[string]any{
				"RecipientName":    "Alpha Aggregators Ltd",
				"PlanName":         "Enterprise Annual Plan",
				"BusinessName":     "Acme Logistics",
				"Reference":        "NXR-REC-20261110-001",
				"AmountPaid":       "₦120,000.00",
				"TotalInvoices":    5000,
				"BillingCycleText": "Yearly",
				"PaymentDate":      "10 November 2026, 12:00 UTC",
				"PaymentChannel":   "Card",
				"NextBillingDate":  "10 November 2027",
				"DashboardURL":     "https://app.nexar.ng/dashboard",
			},
			contains: "RECEIPT DETAILS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderEmail(tt.template, tt.data)
			if err != nil {
				t.Fatalf("RenderEmail(%q) returned error: %v", tt.template, err)
			}
			if !strings.Contains(rendered, tt.contains) {
				t.Errorf("RenderEmail(%q) expected to contain %q, but got:\n%s", tt.template, tt.contains, rendered)
			}
			// Verify shared components are rendered
			if !strings.Contains(rendered, "Nexar") {
				t.Errorf("RenderEmail(%q) missing reusable header 'Nexar'", tt.template)
			}
			if !strings.Contains(rendered, "Nexar. All rights reserved.") {
				t.Errorf("RenderEmail(%q) missing reusable footer copyright", tt.template)
			}
		})
	}
}

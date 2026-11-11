package resend_email

import (
	"einvoice-access-point/templates"
	"fmt"
	"strings"
	"time"
)

// SendInvoiceValidationCompletedEmail sends an email to the business when an invoice passes validation.
func SendInvoiceValidationCompletedEmail(to, businessName, invoiceNumber, irn string, totalAmount float64) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	if businessName == "" {
		businessName = "Valued Business"
	}

	subject := fmt.Sprintf("Invoice Validation Completed: #%s", invoiceNumber)

	irnDisplay := irn
	if strings.TrimSpace(irnDisplay) == "" {
		irnDisplay = "Pending transmission"
	}

	amountDisplay := fmt.Sprintf("NGN %.2f", totalAmount)

	bodyHtml, err := templates.RenderEmail("invoice_validation_completed.html", map[string]any{
		"BusinessName":   businessName,
		"InvoiceNumber":  invoiceNumber,
		"AmountDisplay":  amountDisplay,
		"IRN":            irnDisplay,
		"ValidationTime": time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
	})
	if err != nil {
		return fmt.Errorf("failed to render invoice validation completed email: %w", err)
	}

	return sendEmailInternal(to, subject, bodyHtml)
}

// SendInvoiceValidationFailedEmail sends an email to the business when an invoice validation fails.
func SendInvoiceValidationFailedEmail(to, businessName, invoiceNumber, failureReason string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	if businessName == "" {
		businessName = "Valued Business"
	}

	if failureReason == "" {
		failureReason = "Validation error encountered during schema or FIRS validation."
	}

	subject := fmt.Sprintf("Action Required: Invoice Validation Failed (#%s)", invoiceNumber)

	bodyHtml, err := templates.RenderEmail("invoice_validation_failed.html", map[string]any{
		"BusinessName":  businessName,
		"InvoiceNumber": invoiceNumber,
		"FailureReason": failureReason,
		"Timestamp":     time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
	})
	if err != nil {
		return fmt.Errorf("failed to render invoice validation failed email: %w", err)
	}

	return sendEmailInternal(to, subject, bodyHtml)
}

// SendBulkUploadFinishedEmail sends an email to the business when a bulk invoice batch finishes processing.
func SendBulkUploadFinishedEmail(to, businessName, fileKey string, totalRows, validRows, successful, partial, failed, validationErrors int, duration time.Duration) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	if businessName == "" {
		businessName = "Valued Business"
	}

	cleanFileKey := fileKey
	if idx := strings.LastIndex(cleanFileKey, "/"); idx != -1 {
		cleanFileKey = cleanFileKey[idx+1:]
	}

	subject := fmt.Sprintf("Bulk Upload Completed: %s", cleanFileKey)

	durationStr := duration.Round(time.Millisecond).String()
	if duration <= 0 {
		durationStr = "N/A"
	}

	bodyHtml, err := templates.RenderEmail("bulk_upload_finished.html", map[string]any{
		"BusinessName":     businessName,
		"FileName":         cleanFileKey,
		"TotalRows":        totalRows,
		"Successful":       successful,
		"Partial":          partial,
		"Failed":           failed,
		"ValidRows":        validRows,
		"ValidationErrors": validationErrors,
		"Duration":         durationStr,
	})
	if err != nil {
		return fmt.Errorf("failed to render bulk upload finished email: %w", err)
	}

	return sendEmailInternal(to, subject, bodyHtml)
}

func formatNairaAmount(amount float64) string {
	parts := strings.Split(fmt.Sprintf("%.2f", amount), ".")
	intPart := parts[0]
	decPart := parts[1]

	var result []byte
	n := len(intPart)
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return "₦" + string(result) + "." + decPart
}

// SendSubscriptionExpiringReminderEmail sends an email notifying the business or aggregator that their subscription is expiring soon.
func SendSubscriptionExpiringReminderEmail(to, recipientName, planName string, daysRemaining, remainingInvoices int, nextBillingDate time.Time, planAmount float64, billingCycleDays int, subscriptionID string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	if recipientName == "" {
		recipientName = "Valued Customer"
	}

	if planName == "" {
		planName = "Nexar Standard"
	}

	daysText := fmt.Sprintf("%d days", daysRemaining)
	if daysRemaining == 2 {
		daysText = "2 days (48 hours)"
	} else if daysRemaining == 1 {
		daysText = "1 day (24 hours)"
	} else if daysRemaining <= 0 {
		daysText = "today"
	}

	formattedDate := nextBillingDate.Format("02 January 2006")
	subject := fmt.Sprintf("Your %s plan ends in %s.", planName, daysText)
	if daysRemaining > 14 {
		subject = fmt.Sprintf("Your %s payment is due on %s.", planName, formattedDate)
	}

	billingCycleText := ""
	if billingCycleDays >= 360 {
		billingCycleText = "Yearly"
	} else if billingCycleDays >= 28 && billingCycleDays <= 31 {
		billingCycleText = "Monthly"
	} else if billingCycleDays == 7 {
		billingCycleText = "Weekly"
	} else if billingCycleDays > 0 {
		billingCycleText = fmt.Sprintf("%d Days", billingCycleDays)
	}

	referenceID := ""
	if strings.TrimSpace(subscriptionID) != "" {
		idClean := strings.ReplaceAll(subscriptionID, "-", "")
		if len(idClean) > 6 {
			idClean = idClean[:6]
		}
		referenceID = fmt.Sprintf("NXR-%s-%s", nextBillingDate.Format("2006"), strings.ToUpper(idClean))
	}

	amountDue := ""
	if planAmount > 0 {
		amountDue = formatNairaAmount(planAmount)
	}

	renewalURL := "https://app.nexar.ng/dashboard/subscriptions"

	bodyHtml, err := templates.RenderEmail("subscription_expiring_reminder.html", map[string]any{
		"RecipientName":    recipientName,
		"PlanName":         planName,
		"DaysRemaining":    daysRemaining,
		"DaysText":         daysText,
		"ExpirationDate":   formattedDate,
		"BillingCycleText": billingCycleText,
		"ReferenceID":      referenceID,
		"AmountDue":        amountDue,
		"RemainingInvoices": remainingInvoices,
		"RenewalURL":       renewalURL,
	})
	if err != nil {
		return fmt.Errorf("failed to render subscription expiring reminder email: %w", err)
	}

	return sendEmailInternal(to, subject, bodyHtml)
}

// SendSubscriptionPaymentReceiptEmail sends a payment receipt and confirmation email to the subscriber.
func SendSubscriptionPaymentReceiptEmail(
	to, recipientName, planName, businessName, reference, paymentChannel string,
	amount float64, totalInvoices, billingCycleDays int,
	paymentDate, nextBillingDate time.Time,
) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}

	if recipientName == "" {
		recipientName = "Valued Customer"
	}

	if planName == "" {
		planName = "Nexar Subscription Plan"
	}

	if paymentDate.IsZero() {
		paymentDate = time.Now().UTC()
	}

	if strings.TrimSpace(reference) == "" {
		reference = fmt.Sprintf("NXR-REC-%d", time.Now().Unix())
	}

	amountFormatted := formatNairaAmount(amount)

	billingCycleText := ""
	if billingCycleDays >= 360 {
		billingCycleText = "Yearly"
	} else if billingCycleDays >= 28 && billingCycleDays <= 31 {
		billingCycleText = "Monthly"
	} else if billingCycleDays == 7 {
		billingCycleText = "Weekly"
	} else if billingCycleDays > 0 {
		billingCycleText = fmt.Sprintf("%d Days", billingCycleDays)
	}

	channelDisplay := paymentChannel
	if channelDisplay == "" {
		channelDisplay = "Card / Paystack"
	} else {
		channelDisplay = strings.ReplaceAll(channelDisplay, "_", " ")
		if len(channelDisplay) > 0 {
			channelDisplay = strings.ToUpper(channelDisplay[:1]) + channelDisplay[1:]
		}
	}

	subject := fmt.Sprintf("Payment Receipt: %s Subscription (%s)", planName, reference)

	bodyHtml, err := templates.RenderEmail("subscription_payment_receipt.html", map[string]any{
		"RecipientName":    recipientName,
		"PlanName":         planName,
		"BusinessName":     businessName,
		"Reference":        reference,
		"AmountPaid":       amountFormatted,
		"TotalInvoices":    totalInvoices,
		"BillingCycleText": billingCycleText,
		"PaymentDate":      paymentDate.Format("02 January 2006, 15:04 UTC"),
		"PaymentChannel":   channelDisplay,
		"NextBillingDate":  nextBillingDate.Format("02 January 2006"),
		"DashboardURL":     "https://app.nexar.ng/dashboard",
	})
	if err != nil {
		return fmt.Errorf("failed to render subscription payment receipt email: %w", err)
	}

	return sendEmailInternal(to, subject, bodyHtml)
}


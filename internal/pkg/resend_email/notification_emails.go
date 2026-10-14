package resend_email

import (
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

	bodyHtml := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Invoice Validation Completed</title>
</head>
<body style="margin: 0; padding: 24px; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1e293b;">
  <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);">
    <div style="background-color: #0f172a; padding: 24px; text-align: center;">
      <h1 style="margin: 0; color: #ffffff; font-size: 20px; font-weight: 700; letter-spacing: -0.5px;">Nexar e-Invoice</h1>
    </div>
    <div style="padding: 32px 28px;">
      <div style="display: inline-block; padding: 6px 12px; background-color: #ecfdf5; border: 1px solid #a7f3d0; border-radius: 9999px; color: #047857; font-size: 13px; font-weight: 600; margin-bottom: 20px;">
        &#10003; Validation Completed Successfully
      </div>
      <h2 style="margin: 0 0 12px 0; font-size: 20px; font-weight: 600; color: #0f172a;">Invoice Validated</h2>
      <p style="margin: 0 0 24px 0; font-size: 15px; line-height: 24px; color: #475569;">
        Hello <strong>%s</strong>,<br>
        Your invoice <strong>#%s</strong> has passed validation successfully and is progressing through the electronic invoicing workflow.
      </p>
      <div style="background-color: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0; padding: 20px; margin-bottom: 28px;">
        <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
          <tr>
            <td style="padding: 8px 0; color: #64748b; width: 40%%;">Invoice Number:</td>
            <td style="padding: 8px 0; font-weight: 600; color: #0f172a;">%s</td>
          </tr>
          <tr>
            <td style="padding: 8px 0; color: #64748b;">Total Amount:</td>
            <td style="padding: 8px 0; font-weight: 600; color: #0f172a;">%s</td>
          </tr>
          <tr>
            <td style="padding: 8px 0; color: #64748b;">IRN:</td>
            <td style="padding: 8px 0; font-family: monospace; color: #0f172a; word-break: break-all;">%s</td>
          </tr>
          <tr>
            <td style="padding: 8px 0; color: #64748b;">Validation Time:</td>
            <td style="padding: 8px 0; color: #0f172a;">%s</td>
          </tr>
        </table>
      </div>
      <p style="margin: 0; font-size: 13px; line-height: 20px; color: #94a3b8;">
        This is an automated notification based on your notification preferences. You can update your email preferences anytime in your Business Settings.
      </p>
    </div>
    <div style="background-color: #f1f5f9; padding: 16px 28px; text-align: center; border-top: 1px solid #e2e8f0;">
      <p style="margin: 0; font-size: 12px; color: #64748b;">&copy; %d Nexar e-Invoice Access Point. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`,
		businessName,
		invoiceNumber,
		invoiceNumber,
		amountDisplay,
		irnDisplay,
		time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
		time.Now().UTC().Year(),
	)

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

	bodyHtml := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Invoice Validation Failed</title>
</head>
<body style="margin: 0; padding: 24px; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1e293b;">
  <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);">
    <div style="background-color: #0f172a; padding: 24px; text-align: center;">
      <h1 style="margin: 0; color: #ffffff; font-size: 20px; font-weight: 700; letter-spacing: -0.5px;">Nexar e-Invoice</h1>
    </div>
    <div style="padding: 32px 28px;">
      <div style="display: inline-block; padding: 6px 12px; background-color: #fef2f2; border: 1px solid #fecaca; border-radius: 9999px; color: #dc2626; font-size: 13px; font-weight: 600; margin-bottom: 20px;">
        &#9888; Validation Failed
      </div>
      <h2 style="margin: 0 0 12px 0; font-size: 20px; font-weight: 600; color: #0f172a;">Invoice Validation Issue</h2>
      <p style="margin: 0 0 24px 0; font-size: 15px; line-height: 24px; color: #475569;">
        Hello <strong>%s</strong>,<br>
        Your invoice <strong>#%s</strong> could not be validated. Please review the error details below and make the necessary corrections before re-submitting.
      </p>
      <div style="background-color: #fff1f2; border-radius: 8px; border: 1px solid #ffe4e6; padding: 20px; margin-bottom: 24px;">
        <p style="margin: 0 0 8px 0; font-size: 13px; font-weight: 700; color: #9f1239; text-transform: uppercase; letter-spacing: 0.5px;">Reason for Failure</p>
        <p style="margin: 0; font-size: 14px; line-height: 22px; color: #881337; font-family: monospace; white-space: pre-wrap; word-break: break-word;">%s</p>
      </div>
      <div style="background-color: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0; padding: 16px 20px; margin-bottom: 28px;">
        <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
          <tr>
            <td style="padding: 6px 0; color: #64748b; width: 40%%;">Invoice Number:</td>
            <td style="padding: 6px 0; font-weight: 600; color: #0f172a;">%s</td>
          </tr>
          <tr>
            <td style="padding: 6px 0; color: #64748b;">Timestamp:</td>
            <td style="padding: 6px 0; color: #0f172a;">%s</td>
          </tr>
        </table>
      </div>
      <p style="margin: 0; font-size: 13px; line-height: 20px; color: #94a3b8;">
        This is an automated notification based on your notification preferences. You can update your email preferences anytime in your Business Settings.
      </p>
    </div>
    <div style="background-color: #f1f5f9; padding: 16px 28px; text-align: center; border-top: 1px solid #e2e8f0;">
      <p style="margin: 0; font-size: 12px; color: #64748b;">&copy; %d Nexar e-Invoice Access Point. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`,
		businessName,
		invoiceNumber,
		failureReason,
		invoiceNumber,
		time.Now().UTC().Format("Jan 02, 2006 15:04:05 UTC"),
		time.Now().UTC().Year(),
	)

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

	bodyHtml := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Bulk Upload Finished</title>
</head>
<body style="margin: 0; padding: 24px; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #1e293b;">
  <div style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; border: 1px solid #e2e8f0; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);">
    <div style="background-color: #0f172a; padding: 24px; text-align: center;">
      <h1 style="margin: 0; color: #ffffff; font-size: 20px; font-weight: 700; letter-spacing: -0.5px;">Nexar e-Invoice</h1>
    </div>
    <div style="padding: 32px 28px;">
      <div style="display: inline-block; padding: 6px 12px; background-color: #eff6ff; border: 1px solid #bfdbfe; border-radius: 9999px; color: #1d4ed8; font-size: 13px; font-weight: 600; margin-bottom: 20px;">
        &#9432; Processing Completed
      </div>
      <h2 style="margin: 0 0 12px 0; font-size: 20px; font-weight: 600; color: #0f172a;">Bulk Upload Finished Processing</h2>
      <p style="margin: 0 0 24px 0; font-size: 15px; line-height: 24px; color: #475569;">
        Hello <strong>%s</strong>,<br>
        Your bulk invoice upload batch (<strong>%s</strong>) has finished processing. Here is the summary of the run:
      </p>

      <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; margin-bottom: 24px;">
        <div style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 14px; text-align: center;">
          <div style="font-size: 12px; font-weight: 600; color: #64748b; text-transform: uppercase;">Total Rows</div>
          <div style="font-size: 22px; font-weight: 700; color: #0f172a; margin-top: 4px;">%d</div>
        </div>
        <div style="background-color: #ecfdf5; border: 1px solid #a7f3d0; border-radius: 8px; padding: 14px; text-align: center;">
          <div style="font-size: 12px; font-weight: 600; color: #047857; text-transform: uppercase;">Successful</div>
          <div style="font-size: 22px; font-weight: 700; color: #047857; margin-top: 4px;">%d</div>
        </div>
        <div style="background-color: #fffbeb; border: 1px solid #fde68a; border-radius: 8px; padding: 14px; text-align: center;">
          <div style="font-size: 12px; font-weight: 600; color: #b45309; text-transform: uppercase;">Partially Successful</div>
          <div style="font-size: 22px; font-weight: 700; color: #b45309; margin-top: 4px;">%d</div>
        </div>
        <div style="background-color: #fef2f2; border: 1px solid #fecaca; border-radius: 8px; padding: 14px; text-align: center;">
          <div style="font-size: 12px; font-weight: 600; color: #dc2626; text-transform: uppercase;">Failed / Errors</div>
          <div style="font-size: 22px; font-weight: 700; color: #dc2626; margin-top: 4px;">%d</div>
        </div>
      </div>

      <div style="background-color: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0; padding: 16px 20px; margin-bottom: 28px;">
        <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
          <tr>
            <td style="padding: 6px 0; color: #64748b; width: 45%%;">File Name:</td>
            <td style="padding: 6px 0; font-weight: 600; color: #0f172a; word-break: break-all;">%s</td>
          </tr>
          <tr>
            <td style="padding: 6px 0; color: #64748b;">Valid Schema Rows:</td>
            <td style="padding: 6px 0; color: #0f172a;">%d</td>
          </tr>
          <tr>
            <td style="padding: 6px 0; color: #64748b;">Validation Errors:</td>
            <td style="padding: 6px 0; color: #0f172a;">%d</td>
          </tr>
          <tr>
            <td style="padding: 6px 0; color: #64748b;">Processing Duration:</td>
            <td style="padding: 6px 0; color: #0f172a;">%s</td>
          </tr>
        </table>
      </div>

      <p style="margin: 0; font-size: 13px; line-height: 20px; color: #94a3b8;">
        This is an automated notification based on your notification preferences. You can view full error breakdowns and logs directly in your Bulk Upload portal.
      </p>
    </div>
    <div style="background-color: #f1f5f9; padding: 16px 28px; text-align: center; border-top: 1px solid #e2e8f0;">
      <p style="margin: 0; font-size: 12px; color: #64748b;">&copy; %d Nexar e-Invoice Access Point. All rights reserved.</p>
    </div>
  </div>
</body>
</html>`,
		businessName,
		cleanFileKey,
		totalRows,
		successful,
		partial,
		failed,
		cleanFileKey,
		validRows,
		validationErrors,
		durationStr,
		time.Now().UTC().Year(),
	)

	return sendEmailInternal(to, subject, bodyHtml)
}

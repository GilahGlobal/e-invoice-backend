package zoho

import (
	"einvoice-access-point/pkg/config"
	"einvoice-access-point/pkg/models"
	"einvoice-access-point/pkg/utility"
	"fmt"
)

func UpdateZohoInvoice(accessToken, invoiceID, theIRN, theQrCodeValue string, accConfig models.AccountingPlatformConfig) error {
	var (
		configs = config.GetConfig()
	)

	apiURL := fmt.Sprintf("%s/%s?organization_id=%s", configs.Zoho.ZohoApiUrl, invoiceID, accConfig.OrgID)

	updateData := ZohoUpdateInvoice{
		CustomFields: []ZohoCustomField{
			{
				ApiName: "cf_irn",
				Value:   theIRN,
			},
			{
				ApiName: "cf_qr_code",
				Value:   theQrCodeValue,
			},
		},
	}

	config := utility.RequestConfig{
		URL: apiURL,
		Headers: map[string]string{
			"Authorization": "Zoho-oauthtoken " + accessToken,
			"Content-Type":  "application/json",
		},
		Body: updateData,
	}

	var zohoResp map[string]interface{}

	resp, err := utility.PutRequest(utility.DefaultHTTPClient, config, &zohoResp)
	if err != nil {
		return fmt.Errorf("failed to update Zoho invoice: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("zoho API error: %v, body: %s", resp.StatusCode, string(resp.Body))
	}

	// theErr := updateZohoInvoiceAttachment(configs.Zoho.ZohoApiUrl, accessToken, invoiceID, accConfig.OrgID, theQrCode)
	// if theErr != nil {
	// 	return fmt.Errorf("failed to add attachment invoice: %w", theErr)
	// }

	return nil
}

// func updateZohoInvoiceAttachment(url, accessToken, invoiceID, orgID, qrBase64 string) error {

// 	apiURL := fmt.Sprintf("%s/%s/attachment?organization_id=%s", url, invoiceID, orgID)
// 	fmt.Println("Zoho Invoice Update URL:", apiURL)

// 	qrBytes, err := base64.StdEncoding.DecodeString(qrBase64)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode QR base64: %w", err)
// 	}

// 	var buf bytes.Buffer
// 	writer := multipart.NewWriter(&buf)

// 	fileField, err := writer.CreateFormFile("attachment", filepath.Base("qrcode.png"))
// 	if err != nil {
// 		return fmt.Errorf("failed to create form file: %w", err)
// 	}
// 	if _, err = fileField.Write(qrBytes); err != nil {
// 		return fmt.Errorf("failed to write file bytes: %w", err)
// 	}

// 	_ = writer.WriteField("can_send_in_mail", "true")

// 	if err := writer.Close(); err != nil {
// 		return fmt.Errorf("failed to close writer: %w", err)
// 	}

// 	req, err := http.NewRequest("POST", apiURL, &buf)
// 	if err != nil {
// 		return fmt.Errorf("failed to create request: %w", err)
// 	}
// 	req.Header.Set("Authorization", "Zoho-oauthtoken "+accessToken)
// 	req.Header.Set("Content-Type", writer.FormDataContentType())

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("failed to send request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		return fmt.Errorf("zoho API error: %v", resp.Status)
// 	}

// 	fmt.Println("Attachment uploaded successfully ✅")
// 	return nil
// }

package cloudinary

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"einvoice-access-point/internal/config"
)

type uploadResponse struct {
	SecureURL string `json:"secure_url"`
	URL       string `json:"url"`
	Error     struct {
		Message string `json:"message"`
	} `json:"error"`
}

// UploadBMPBase64 uploads a base64-encoded BMP payload to Cloudinary and returns the hosted URL.
func UploadBMPBase64(base64BMP, publicID string) (string, error) {
	cfg := config.GetConfig()
	if cfg == nil {
		return "", fmt.Errorf("cloudinary config is not available")
	}
	if cfg.Cloudinary.CloudName == "" || cfg.Cloudinary.APIKey == "" || cfg.Cloudinary.APISecret == "" {
		return "", fmt.Errorf("cloudinary credentials are not configured")
	}

	rawBMP, err := decodeBase64Payload(base64BMP)
	if err != nil {
		return "", err
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	params := map[string]string{
		"timestamp": timestamp,
	}
	if publicID != "" {
		params["public_id"] = publicID
	}
	if folder := strings.TrimSpace(cfg.Cloudinary.Folder); folder != "" {
		params["folder"] = folder
	}

	signature := signParams(params, cfg.Cloudinary.APISecret)
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cfg.Cloudinary.CloudName)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, key := range sortedKeys(params) {
		if err := writer.WriteField(key, params[key]); err != nil {
			return "", err
		}
	}
	if err := writer.WriteField("api_key", cfg.Cloudinary.APIKey); err != nil {
		return "", err
	}
	if err := writer.WriteField("signature", signature); err != nil {
		return "", err
	}

	part, err := writer.CreateFormFile("file", fileName(publicID))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, bytes.NewReader(rawBMP)); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed uploadResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse cloudinary response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		msg := parsed.Error.Message
		if msg == "" {
			msg = strings.TrimSpace(string(respBody))
		}
		return "", fmt.Errorf("cloudinary upload failed: %s", msg)
	}

	if parsed.SecureURL != "" {
		return parsed.SecureURL, nil
	}
	if parsed.URL != "" {
		return parsed.URL, nil
	}

	return "", fmt.Errorf("cloudinary upload succeeded but returned no url")
}

func decodeBase64Payload(base64Str string) ([]byte, error) {
	if base64Str == "" {
		return nil, fmt.Errorf("empty image payload")
	}

	if strings.Contains(base64Str, ",") {
		parts := strings.SplitN(base64Str, ",", 2)
		base64Str = parts[1]
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func signParams(params map[string]string, secret string) string {
	keys := sortedKeys(params)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, fmt.Sprintf("%s=%s", key, params[key]))
	}

	hasher := sha1.New()
	hasher.Write([]byte(strings.Join(values, "&") + secret))
	return hex.EncodeToString(hasher.Sum(nil))
}

func sortedKeys(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fileName(publicID string) string {
	if publicID == "" {
		return fmt.Sprintf("invoice-bmp-%d.bmp", time.Now().UnixNano())
	}
	return publicID + ".bmp"
}

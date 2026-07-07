package zoho

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	ZOHO_REDIRECT_URL = "http://localhost:8091/api/v1/zoho/callback"
)

// GenerateAuthURL generates the authorization URL for Step 2: Generating Grant Token.
func GenerateAuthURL(state, redirectURI string) string {
	scope := "ZohoInvoice.fullaccess.all"
	authURL := "https://accounts.zoho.com/oauth/v2/auth"

	params := url.Values{}
	params.Add("scope", scope)
	params.Add("client_id", "ZOHO_CLIENT_ID")
	params.Add("state", state)
	params.Add("response_type", "code")
	params.Add("redirect_uri", redirectURI)
	params.Add("access_type", "offline")

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// ExchangeCodeForTokens performs Step 3: Generate Access and Refresh Token.
func ExchangeCodeForTokens(code, clientID, clientSecret string) (*TokenResponse, error) {
	tokenURL := "https://accounts.zoho.com/oauth/v2/token"

	params := url.Values{}
	params.Add("code", code)
	params.Add("client_id", clientID)
	params.Add("client_secret", clientSecret)
	params.Add("redirect_uri", ZOHO_REDIRECT_URL)
	params.Add("grant_type", "authorization_code")

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, err
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("error from Zoho: %s", tokenResp.Error)
	}

	return &tokenResp, nil
}

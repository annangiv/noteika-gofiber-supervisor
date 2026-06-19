package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type OAuthUserProfile struct {
	Provider   string
	ProviderID string
	Email      string
	Name       string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

// Check if provider is configured for mock mode
func IsMockProvider(provider string) bool {
	clientID := os.Getenv(strings.ToUpper(provider) + "_OAUTH_CLIENT_ID")
	return clientID == "" || clientID == "mock"
}

// GetAuthURL generates authorization URL. For mock, redirects to local mock endpoint.
func GetAuthURL(provider string, state string) (string, error) {
	if IsMockProvider(provider) {
		// Redirect to mock auth URL
		return fmt.Sprintf("/oauth/mock/authorize?provider=%s&state=%s", provider, state), nil
	}

	config, err := getOauthConfig(provider)
	if err != nil {
		return "", err
	}

	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCodeAndFetchProfile exchanges the authorization code for a profile.
func ExchangeCodeAndFetchProfile(provider string, code string, redirectURI string) (OAuthUserProfile, error) {
	if IsMockProvider(provider) || strings.HasPrefix(code, "mock_") {
		return exchangeMockCode(provider, code)
	}

	config, err := getOauthConfig(provider)
	if err != nil {
		return OAuthUserProfile{}, err
	}

	// Exchange code for token
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return OAuthUserProfile{}, fmt.Errorf("token exchange failed: %w", err)
	}

	switch provider {
	case "github":
		return fetchGithubProfile(token.AccessToken)
	case "google":
		return fetchGoogleProfile(token.AccessToken)
	default:
		return OAuthUserProfile{}, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func getOauthConfig(provider string) (*oauth2.Config, error) {
	prefix := strings.ToUpper(provider) + "_OAUTH_"
	clientID := os.Getenv(prefix + "CLIENT_ID")
	clientSecret := os.Getenv(prefix + "CLIENT_SECRET")
	redirectURL := os.Getenv(prefix + "REDIRECT_URI")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing oauth credentials for %s", provider)
	}

	var endpoint oauth2.Endpoint
	var scopes []string

	switch provider {
	case "github":
		endpoint = github.Endpoint
		scopes = []string{"read:user", "user:email"}
	case "google":
		endpoint = google.Endpoint
		scopes = []string{"openid", "email", "profile"}
	default:
		return nil, fmt.Errorf("unknown oauth provider: %s", provider)
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     endpoint,
		Scopes:       scopes,
	}, nil
}

// Stateless mock login: authorization code is base64(JSON(profile))
func exchangeMockCode(provider string, code string) (OAuthUserProfile, error) {
	if !strings.HasPrefix(code, "mock_") {
		return OAuthUserProfile{}, fmt.Errorf("invalid mock authorization code")
	}

	encodedPayload := strings.TrimPrefix(code, "mock_")
	decodedBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return OAuthUserProfile{}, fmt.Errorf("failed to decode mock payload: %w", err)
	}

	var payload struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(decodedBytes, &payload); err != nil {
		return OAuthUserProfile{}, fmt.Errorf("invalid mock payload: %w", err)
	}

	// Generate a deterministic provider ID based on email
	providerID := fmt.Sprintf("mock-id-%s", payload.Email)

	return OAuthUserProfile{
		Provider:   provider,
		ProviderID: providerID,
		Email:      payload.Email,
		Name:       payload.Name,
	}, nil
}

func fetchGithubProfile(token string) (OAuthUserProfile, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Get user profile
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-fiber-supervisor")

	resp, err := client.Do(req)
	if err != nil {
		return OAuthUserProfile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OAuthUserProfile{}, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var ghUser struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return OAuthUserProfile{}, err
	}

	email := ghUser.Email
	if email == "" {
		// 2. Fetch primary verified email if public email is empty
		email, err = fetchGithubEmails(client, token)
		if err != nil {
			return OAuthUserProfile{}, fmt.Errorf("failed to fetch user emails: %w", err)
		}
	}

	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}

	return OAuthUserProfile{
		Provider:   "github",
		ProviderID: fmt.Sprintf("%d", ghUser.ID),
		Email:      email,
		Name:       name,
	}, nil
}

func fetchGithubEmails(client *http.Client, token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-fiber-supervisor")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails api returned status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no primary verified email found")
}

func fetchGoogleProfile(token string) (OAuthUserProfile, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return OAuthUserProfile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return OAuthUserProfile{}, fmt.Errorf("google api returned status %d: %s", resp.StatusCode, string(body))
	}

	var gUser struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return OAuthUserProfile{}, err
	}

	if !gUser.EmailVerified {
		return OAuthUserProfile{}, fmt.Errorf("google email is not verified")
	}

	name := gUser.Name
	if name == "" {
		name = strings.Split(gUser.Email, "@")[0]
	}

	return OAuthUserProfile{
		Provider:   "google",
		ProviderID: gUser.Sub,
		Email:      gUser.Email,
		Name:       name,
	}, nil
}

// ServeMockAuthorizeScreen writes the mock authorization form directly.
func ServeMockAuthorizeScreen(w http.ResponseWriter, req *http.Request) {
	provider := req.URL.Query().Get("provider")
	state := req.URL.Query().Get("state")

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>Developer Mock Authorization</title>
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
			body {
				font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
				background-color: #0d1117;
				color: #c9d1d9;
				display: flex;
				justify-content: center;
				align-items: center;
				height: 100vh;
				margin: 0;
			}
			.card {
				background-color: #161b22;
				border: 1px solid #30363d;
				border-radius: 12px;
				padding: 32px;
				width: 360px;
				box-shadow: 0 8px 24px rgba(0,0,0,0.5);
			}
			h2 { margin-top: 0; color: #58a6ff; font-weight: 500; }
			label { display: block; margin: 16px 0 6px; font-size: 14px; color: #8b949e; }
			input {
				width: 100%%;
				padding: 10px;
				background-color: #0d1117;
				border: 1px solid #30363d;
				border-radius: 6px;
				color: #fff;
				font-size: 14px;
				box-sizing: border-box;
			}
			input:focus { border-color: #58a6ff; outline: none; }
			button {
				width: 100%%;
				padding: 12px;
				background-color: #238636;
				color: white;
				border: none;
				border-radius: 6px;
				font-size: 16px;
				font-weight: 600;
				margin-top: 24px;
				cursor: pointer;
				transition: background 0.2s;
			}
			button:hover { background-color: #2ea44f; }
			.info { font-size: 12px; color: #8b949e; margin-top: 16px; text-align: center; line-height: 1.4; }
		</style>
	</head>
	<body>
		<div class="card">
			<h2>Mock OAuth Login</h2>
			<div style="font-size: 14px; color: #8b949e; margin-bottom: 20px;">
				Simulating login for <strong>%s</strong>
			</div>
			<form action="/oauth/mock/submit" method="POST">
				<input type="hidden" name="provider" value="%s">
				<input type="hidden" name="state" value="%s">

				<label for="name">Full Name</label>
				<input type="text" id="name" name="name" value="Developer User" required>

				<label for="email">Email Address</label>
				<input type="email" id="email" name="email" value="dev-user@example.com" required>

				<button type="submit">Authorize Developer App</button>
			</form>
			<div class="info">
				This bypasses external APIs and generates a secure local session. Use any test account name or email.
			</div>
		</div>
	</body>
	</html>
	`, provider, provider, state)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// HandleMockSubmit processes the mock form submission and redirects back to the callback URL.
func HandleMockSubmit(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := req.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	provider := req.FormValue("provider")
	state := req.FormValue("state")
	name := req.FormValue("name")
	email := req.FormValue("email")

	if name == "" || email == "" {
		http.Error(w, "Name and Email are required", http.StatusBadRequest)
		return
	}

	// Create a stateless JSON payload representing user profile
	profile := struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{
		Email: email,
		Name:  name,
	}

	jsonBytes, _ := json.Marshal(profile)
	encodedPayload := base64.URLEncoding.EncodeToString(jsonBytes)
	mockCode := fmt.Sprintf("mock_%s", encodedPayload)

	// Redirect back to callback endpoint
	callbackURL := fmt.Sprintf("/auth/%s/callback?code=%s&state=%s", provider, mockCode, url.QueryEscape(state))
	http.Redirect(w, req, callbackURL, http.StatusFound)
}

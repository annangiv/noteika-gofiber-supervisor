package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// GetAuthURL generates authorization URL.
func GetAuthURL(provider string, state string) (string, error) {
	config, err := getOauthConfig(provider)
	if err != nil {
		return "", err
	}

	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCodeAndFetchProfile exchanges the authorization code for a profile.
func ExchangeCodeAndFetchProfile(provider string, code string, redirectURI string) (OAuthUserProfile, error) {
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

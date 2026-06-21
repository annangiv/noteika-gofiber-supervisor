package web

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"my-app/actor"
	"my-app/db"
)

// GoogleNativeLogin handles verified OAuth ID tokens from native iOS/Android clients.
func (h *AuthHandler) GoogleNativeLogin(c *fiber.Ctx) error {
	var input struct {
		IDToken string `json:"id_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	idToken := strings.TrimSpace(input.IDToken)
	if idToken == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing id_token"})
	}

	// Validate the ID token with Google's tokeninfo API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		log.Printf("[AuthHandler] Google tokeninfo request failed: %v", err)
		return c.Status(502).JSON(fiber.Map{"error": "failed to contact Google token verification endpoint"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[AuthHandler] Google tokeninfo rejected token: %s", string(body))
		return c.Status(401).JSON(fiber.Map{"error": "invalid google id token"})
	}

	var gUser struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified string `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to parse google verification response"})
	}

	if gUser.Email == "" {
		return c.Status(400).JSON(fiber.Map{"error": "google token did not contain an email address"})
	}

	log.Printf("[AuthHandler] Google Native Login verified: %s (%s)", gUser.Name, gUser.Email)

	profile := OAuthUserProfile{
		Provider:   "google",
		ProviderID: gUser.Sub,
		Email:      gUser.Email,
		Name:       gUser.Name,
	}

	userID, user, err := h.upsertUserFromProfile(profile)
	if err != nil {
		log.Printf("[AuthHandler] upsertUserFromProfile failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to save user info"})
	}

	sessionID := uuid.New().String()
	now := time.Now().Unix()
	session := db.Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now + SessionTTLSecs,
	}
	if _, err := h.gateway.Send(actor.TypeSaveSession, actor.SaveSessionPayload{Session: session}, 5*time.Second); err != nil {
		log.Printf("[AuthHandler] SaveSession failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to create session"})
	}

	// Set session cookie for web/hybrid environments if needed
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(SessionTTLSecs * time.Second),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	billing := BillingHandler{gateway: h.gateway}
	captureCount, _ := billing.activeCaptureCount(userID)

	return c.JSON(fiber.Map{
		"token":         sessionID,
		"user": fiber.Map{
			"id":             userID,
			"email":          gUser.Email,
			"oauth_provider": user.OAuthProvider,
			"full_name":      gUser.Name,
			"pro_access":     billing.userHasProAccess(user),
			"capture_count":  captureCount,
			"capture_limit":  db.FreeCaptureLimit(),
		},
	})
}

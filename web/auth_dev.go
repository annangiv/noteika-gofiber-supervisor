package web

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

func IsDevelopment() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENVIRONMENT")), "development")
}

// DevLogin creates a session without external OAuth — development / mobile emulator only.
func (h *AuthHandler) DevLogin(c *fiber.Ctx) error {
	if !IsDevelopment() {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}

	var input struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	email := strings.TrimSpace(input.Email)
	name := strings.TrimSpace(input.Name)
	if email == "" {
		email = "dev-user@example.com"
	}
	if name == "" {
		name = "Developer User"
	}

	profile := OAuthUserProfile{
		Provider:   "dev",
		ProviderID: email,
		Email:      email,
		Name:       name,
	}

	userID, user, err := h.upsertUserFromProfile(profile)
	if err != nil {
		log.Printf("[AuthHandler] DevLogin upsert failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to save user"})
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
		return c.Status(500).JSON(fiber.Map{"error": "failed to create session"})
	}

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
		"id":             userID,
		"email":          email,
		"oauth_provider": user.OAuthProvider,
		"full_name":      name,
		"pro_access":     billing.userHasProAccess(user),
		"capture_count":  captureCount,
		"capture_limit":  db.FreeCaptureLimit(),
	})
}

func (h *AuthHandler) upsertUserFromProfile(userProfile OAuthUserProfile) (string, db.User, error) {
	emailHash := utils.HashEmail(userProfile.Email)
	encryptedEmail, err := utils.EncryptEmail(userProfile.Email)
	if err != nil {
		return "", db.User{}, err
	}

	var userID string
	var existingUser db.User
	hashPayload := actor.GetUserByEmailHashPayload{Hash: emailHash}
	existingRes, err := h.gateway.Send(actor.TypeGetUserByEmailHash, hashPayload, 5*time.Second)
	if err == nil {
		if u, ok := existingRes.(db.User); ok {
			existingUser = u
			userID = u.ID
		}
	}
	if userID == "" {
		userID = uuid.New().String()
	}

	now := time.Now().Unix()
	createdAt := now
	if existingUser.ID != "" {
		createdAt = existingUser.CreatedAt
	}

	user := db.User{
		ID:                  userID,
		EmailHash:           emailHash,
		EncryptedEmail:      encryptedEmail,
		OAuthProvider:       userProfile.Provider,
		OAuthID:             userProfile.ProviderID,
		FullName:            userProfile.Name,
		Tier:                "free",
		SearchMinSimilarity: existingUser.SearchMinSimilarity,
		CreatedAt:           createdAt,
		UpdatedAt:           now,
	}
	if existingUser.Tier != "" {
		user.Tier = existingUser.Tier
	}
	if isOwnerEmail(userProfile.Email) {
		user.Tier = db.TierPro
	}
	user.StripeCustomerID = existingUser.StripeCustomerID
	user.EncryptionSalt = existingUser.EncryptionSalt

	if _, err := h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second); err != nil {
		return "", db.User{}, err
	}
	return userID, user, nil
}

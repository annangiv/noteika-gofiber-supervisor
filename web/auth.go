package web

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

const (
	SessionCookieName = "keller_session"
	SessionTTLSecs    = 30 * 24 * 60 * 60 // 30 days
)

type AuthHandler struct {
	gateway *actor.ActorGateway
}

func NewAuthHandler(gateway *actor.ActorGateway) *AuthHandler {
	return &AuthHandler{gateway: gateway}
}

// Redirect to OAuth provider
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider != "github" && provider != "google" {
		return c.Status(400).SendString("Unknown OAuth provider")
	}

	state := utils.GenerateRandomHex(16)
	expiresAt := time.Now().Add(10 * time.Minute).Unix()

	// Persist state in BadgerDB via VaultActor
	payload := actor.SaveOauthStatePayload{
		State:     state,
		Provider:  provider,
		ExpiresAt: expiresAt,
	}
	_, err := h.gateway.Send(actor.TypeSaveOauthState, payload, 5*time.Second)
	if err != nil {
		log.Printf("[AuthHandler] SaveOauthState failed: %v", err)
		return c.Status(500).SendString("Database error initiating authentication")
	}

	authURL, err := GetAuthURL(provider, state)
	if err != nil {
		return c.Status(500).SendString("Failed to build authorization URL")
	}

	return c.Redirect(authURL)
}

// Handle callback from OAuth provider
func (h *AuthHandler) Callback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider != "github" && provider != "google" {
		return c.Status(400).SendString("Unknown OAuth provider")
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Status(400).SendString("Missing code or state parameters")
	}

	// Consume/verify state from DB atomically
	takePayload := actor.TakeOauthStatePayload{State: state}
	res, err := h.gateway.Send(actor.TypeTakeOauthState, takePayload, 5*time.Second)
	if err != nil {
		log.Printf("[AuthHandler] TakeOauthState failed: %v", err)
		return c.Status(400).SendString("Invalid or expired OAuth state")
	}

	storedProvider, ok := res.(string)
	if !ok || storedProvider != provider {
		return c.Status(400).SendString("OAuth state validation failed")
	}

	// Fetch user profile from provider
	userProfile, err := ExchangeCodeAndFetchProfile(provider, code, c.BaseURL()+c.Path())
	if err != nil {
		log.Printf("[AuthHandler] ExchangeCodeAndFetchProfile failed: %v", err)
		return c.Status(502).SendString("Failed to fetch user profile from provider: " + err.Error())
	}

	emailHash := utils.HashEmail(userProfile.Email)
	encryptedEmail, err := utils.EncryptEmail(userProfile.Email)
	if err != nil {
		return c.Status(500).SendString("Encryption failed")
	}

	// Check if user already exists (preserve id, created_at, and settings on re-login)
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

	// Upsert User
	_, err = h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second)
	if err != nil {
		log.Printf("[AuthHandler] UpsertUser failed: %v", err)
		return c.Status(500).SendString("Failed to save user profile")
	}

	// Create Session
	sessionID := uuid.New().String()
	session := db.Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now + SessionTTLSecs,
	}

	_, err = h.gateway.Send(actor.TypeSaveSession, actor.SaveSessionPayload{Session: session}, 5*time.Second)
	if err != nil {
		log.Printf("[AuthHandler] SaveSession failed: %v", err)
		return c.Status(500).SendString("Failed to create user session")
	}

	// Set session cookie
	cookie := &fiber.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(SessionTTLSecs * time.Second),
		HTTPOnly: true,
		SameSite: "Lax",
	}
	c.Cookie(cookie)

	return c.Redirect("/notes")
}

// Log out user
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := c.Cookies(SessionCookieName)
	if sessionID != "" {
		// Delete session from DB via actor
		_, _ = h.gateway.Send(actor.TypeDeleteSession, actor.DeleteSessionPayload{ID: sessionID}, 5*time.Second)
	}

	// Clear cookie
	cookie := &fiber.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	}
	c.Cookie(cookie)

	return c.Redirect("/login")
}

// Decrypted user profile view for frontend API
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(412).JSON(fiber.Map{"error": "precondition failed"}) // Unauthenticated
	}

	res, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to retrieve user"})
	}

	user, ok := res.(db.User)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "user profile not found"})
	}

	decryptedEmail, err := utils.DecryptEmail(user.EncryptedEmail)
	if err != nil {
		log.Printf("[AuthHandler] DecryptEmail failed: %v", err)
		decryptedEmail = "unknown"
	}

	billing := BillingHandler{gateway: h.gateway}
	captureCount, _ := billing.activeCaptureCount(userID)
	proAccess := billing.userHasProAccess(user)

	return c.JSON(fiber.Map{
		"id":                    user.ID,
		"email":                 decryptedEmail,
		"oauth_provider":        user.OAuthProvider,
		"full_name":             user.FullName,
		"tier":                  user.Tier,
		"pro_access":            proAccess,
		"capture_count":         captureCount,
		"capture_limit":         db.FreeCaptureLimit(),
		"stripe_enabled":        StripeEnabled(),
		"has_stripe_customer":   user.StripeCustomerID != "",
		"created_at":            user.CreatedAt,
		"search_min_similarity": db.EffectiveSearchMinSimilarity(user.SearchMinSimilarity),
	})
}

package web

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

type AccountHandler struct {
	gateway *actor.ActorGateway
}

func NewAccountHandler(gateway *actor.ActorGateway) *AccountHandler {
	return &AccountHandler{gateway: gateway}
}

type exportCapture struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	SourceURL string `json:"source_url"`
	Type      string `json:"type"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type exportPayload struct {
	ExportedAt string          `json:"exported_at"`
	User       exportUser      `json:"user"`
	Captures   []exportCapture `json:"captures"`
}

type exportUser struct {
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Tier      string `json:"tier"`
	CreatedAt int64  `json:"created_at"`
}

// Export downloads all active captures as JSON.
func (h *AccountHandler) Export(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	userRes, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user profile"})
	}
	user, ok := userRes.(db.User)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	email, err := utils.DecryptEmail(user.EncryptedEmail)
	if err != nil {
		email = "unknown"
	}

	listRes, err := h.gateway.Send(actor.TypeListCaptures, actor.ListCapturesPayload{
		UserID:        userID,
		ProjectFilter: "",
	}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to export captures"})
	}

	captures, ok := listRes.([]db.Capture)
	if !ok || captures == nil {
		captures = []db.Capture{}
	}

	exported := make([]exportCapture, 0, len(captures))
	for _, cap := range captures {
		exported = append(exported, exportCapture{
			ID:        cap.ID,
			Project:   cap.Project,
			Title:     cap.Title,
			Body:      cap.Body,
			SourceURL: cap.SourceURL,
			Type:      cap.Type,
			CreatedAt: cap.CreatedAt,
			UpdatedAt: cap.UpdatedAt,
		})
	}

	payload := exportPayload{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		User: exportUser{
			Email:     email,
			FullName:  user.FullName,
			Tier:      user.Tier,
			CreatedAt: user.CreatedAt,
		},
		Captures: exported,
	}

	filename := "noteika-export.json"
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return json.NewEncoder(c).Encode(payload)
}

// DeleteAccount permanently removes the user and all captures.
func (h *AccountHandler) DeleteAccount(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	sessionID := c.Cookies(SessionCookieName)
	if sessionID != "" {
		_, _ = h.gateway.Send(actor.TypeDeleteSession, actor.DeleteSessionPayload{ID: sessionID}, 5*time.Second)
	}

	_, err := h.gateway.Send(actor.TypeDeleteUser, actor.DeleteUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		log.Printf("[AccountHandler] DeleteUser failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete account"})
	}

	cookie := &fiber.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	}
	c.Cookie(cookie)

	return c.JSON(fiber.Map{"message": "account deleted successfully"})
}

package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	ID         string   `json:"id"`
	ProjectID  string   `json:"project_id"`
	Ciphertext string   `json:"ciphertext,omitempty"`
	Title      string   `json:"title,omitempty"`
	Body       string   `json:"body,omitempty"`
	SourceURL  string   `json:"source_url,omitempty"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`
	CreatedAt  int64    `json:"created_at"`
	UpdatedAt  int64    `json:"updated_at"`
}

type exportProject struct {
	ID         string `json:"id"`
	Ciphertext string `json:"ciphertext"`
	CreatedAt  int64  `json:"created_at"`
}

type exportPayload struct {
	ExportedAt string          `json:"exported_at"`
	User       exportUser      `json:"user"`
	Projects   []exportProject `json:"projects"`
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
		exp := exportCapture{
			ID:        cap.ID,
			ProjectID: cap.ProjectID,
			Type:      cap.Type,
			CreatedAt: cap.CreatedAt,
			UpdatedAt: cap.UpdatedAt,
		}
		if cap.IsEncrypted() {
			exp.Ciphertext = base64.StdEncoding.EncodeToString(cap.Ciphertext)
		} else {
			exp.Title = cap.Title
			exp.Body = cap.Body
			exp.SourceURL = cap.SourceURL
			exp.Tags = cap.Tags
		}
		exported = append(exported, exp)
	}

	projectsRes, err := h.gateway.Send(actor.TypeListProjects, actor.ListProjectsPayload{UserID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to export projects"})
	}
	projects, ok := projectsRes.([]db.Project)
	if !ok {
		projects = []db.Project{}
	}
	exportedProjects := make([]exportProject, 0, len(projects))
	for _, p := range projects {
		exportedProjects = append(exportedProjects, exportProject{
			ID:         p.ID,
			Ciphertext: base64.StdEncoding.EncodeToString(p.Ciphertext),
			CreatedAt:  p.CreatedAt,
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
		Projects: exportedProjects,
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

// UpdateSettings saves user preferences (search sensitivity, etc.).
func (h *AccountHandler) UpdateSettings(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input struct {
		SearchMinSimilarity *float32 `json:"search_min_similarity"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if input.SearchMinSimilarity == nil {
		return c.Status(400).JSON(fiber.Map{"error": "search_min_similarity is required"})
	}

	value := *input.SearchMinSimilarity
	if value < db.MinSearchMinSimilarity || value > db.MaxSearchMinSimilarity {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("search_min_similarity must be between %.0f and %.0f",
				db.MinSearchMinSimilarity, db.MaxSearchMinSimilarity),
		})
	}

	res, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}
	user, ok := res.(db.User)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	user.SearchMinSimilarity = value
	user.UpdatedAt = time.Now().Unix()

	if _, err := h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second); err != nil {
		log.Printf("[AccountHandler] UpdateSettings UpsertUser failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to save settings"})
	}

	return c.JSON(fiber.Map{
		"search_min_similarity": value,
	})
}

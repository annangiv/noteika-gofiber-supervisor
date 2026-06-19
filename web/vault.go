package web

import (
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

type VaultHandler struct {
	gateway *actor.ActorGateway
}

func NewVaultHandler(gateway *actor.ActorGateway) *VaultHandler {
	return &VaultHandler{gateway: gateway}
}

// GetSalt returns (or creates) the per-user salt for vault key derivation.
func (h *VaultHandler) GetSalt(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	res, err := h.gateway.Send(actor.TypeGetUser, actor.GetUserPayload{ID: userID}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user"})
	}
	user, ok := res.(db.User)
	if !ok {
		return c.Status(500).JSON(fiber.Map{"error": "invalid user record"})
	}

	if len(user.EncryptionSalt) == 0 {
		salt, err := utils.RandomBytes(16)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to generate salt"})
		}
		user.EncryptionSalt = salt
		user.UpdatedAt = time.Now().Unix()
		if _, err := h.gateway.Send(actor.TypeUpsertUser, actor.UpsertUserPayload{User: user}, 5*time.Second); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to save vault salt"})
		}
	}

	return c.JSON(fiber.Map{
		"salt": base64.StdEncoding.EncodeToString(user.EncryptionSalt),
	})
}

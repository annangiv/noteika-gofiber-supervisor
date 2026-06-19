package web

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"my-app/actor"
	"my-app/supervisor"
)

type DebugHandler struct {
	supervisor *supervisor.Supervisor
	registry   *actor.ActorRegistry
	gateway    *actor.ActorGateway
}

func NewDebugHandler(sup *supervisor.Supervisor, reg *actor.ActorRegistry, gateway *actor.ActorGateway) *DebugHandler {
	return &DebugHandler{
		supervisor: sup,
		registry:   reg,
		gateway:    gateway,
	}
}

// Trigger a panic/crash inside the VaultActor
func (h *DebugHandler) Crash(c *fiber.Ctx) error {
	// Send TypeDebugCrash message. The actor will panic, the gateway receives the failure response
	_, err := h.gateway.Send(actor.TypeDebugCrash, nil, 3*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status": "crashed",
			"error":  err.Error(),
			"info":   "VaultActor crashed successfully. Check server logs to see the Supervisor restart it.",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Send crash message but actor did not panic.",
	})
}

// Get supervisor stats for the dashboard telemetry
func (h *DebugHandler) Stats(c *fiber.Ctx) error {
	restarts, status := h.supervisor.GetStats()
	mailbox := h.registry.GetMailbox()

	mailboxAddr := "nil"
	if mailbox != nil {
		mailboxAddr = fmt.Sprintf("%p", mailbox)
	}

	return c.JSON(fiber.Map{
		"restarts_count": restarts,
		"status":         status,
		"mailbox_addr":   mailboxAddr,
		"time":           time.Now().Format(time.RFC3339),
	})
}

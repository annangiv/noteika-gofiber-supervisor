package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func GetData(c *fiber.Ctx) error {
	// Test by creating some sample data.
	time.Sleep(100 * time.Millisecond)
	return c.JSON(fiber.Map{
		"data":      []string{"item1", "item2", "item3"},
		"timestamp": time.Now().Unix(),
	})
}

func PostData(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	return c.JSON(fiber.Map{
		"message":  "Data received",
		"received": payload,
	})
}

package web

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"my-app/actor"
	"my-app/db"
)

type NotesHandler struct {
	gateway *actor.ActorGateway
}

func NewNotesHandler(gateway *actor.ActorGateway) *NotesHandler {
	return &NotesHandler{gateway: gateway}
}

// List all notes for the logged-in user
func (h *NotesHandler) List(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	payload := actor.ListNotesPayload{UserID: userID}
	res, err := h.gateway.Send(actor.TypeListNotes, payload, 5*time.Second)
	if err != nil {
		log.Printf("[NotesHandler] ListNotes failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve notes"})
	}

	notes, ok := res.([]db.Note)
	if !ok {
		notes = []db.Note{} // Return empty list instead of null
	}

	return c.JSON(notes)
}

// Create a new note
func (h *NotesHandler) Create(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if input.Title == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Note title is required"})
	}

	note := db.Note{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     input.Title,
		Content:   input.Content,
		CreatedAt: time.Now().Unix(),
	}

	payload := actor.SaveNotePayload{Note: note}
	_, err := h.gateway.Send(actor.TypeSaveNote, payload, 5*time.Second)
	if err != nil {
		log.Printf("[NotesHandler] SaveNote failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create note"})
	}

	return c.Status(201).JSON(note)
}

// Delete a note
func (h *NotesHandler) Delete(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	noteID := c.Params("id")
	if noteID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Note ID is required"})
	}

	payload := actor.DeleteNotePayload{
		UserID: userID,
		ID:     noteID,
	}

	_, err := h.gateway.Send(actor.TypeDeleteNote, payload, 5*time.Second)
	if err != nil {
		log.Printf("[NotesHandler] DeleteNote failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete note or note not found"})
	}

	return c.JSON(fiber.Map{"message": "Note deleted successfully"})
}

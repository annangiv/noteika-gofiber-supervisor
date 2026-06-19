package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"my-app/actor"
	"my-app/db"
	"my-app/utils"
)

type CapturesHandler struct {
	gateway *actor.ActorGateway
}

func NewCapturesHandler(gateway *actor.ActorGateway) *CapturesHandler {
	return &CapturesHandler{gateway: gateway}
}

// Generate an automatic title from capture body text
func generateAutoTitle(body string) string {
	cleaned := strings.TrimSpace(body)
	if cleaned == "" {
		return "Untitled Capture"
	}

	// Take the first non-empty line
	lines := strings.Split(cleaned, "\n")
	var firstLine string
	for _, l := range lines {
		lTrimmed := strings.TrimSpace(l)
		if lTrimmed != "" {
			firstLine = lTrimmed
			break
		}
	}

	if firstLine == "" {
		firstLine = cleaned
	}

	// Truncate to ~80 chars
	if len(firstLine) > 80 {
		// Try to truncate at a word boundary
		subStr := firstLine[:80]
		lastSpace := strings.LastIndex(subStr, " ")
		if lastSpace > 60 {
			return firstLine[:lastSpace] + "..."
		}
		return subStr + "..."
	}

	return firstLine
}

// Classify the capture body text into note, link, qa, or code
func classifyContentType(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "note"
	}

	// 1. Link Detection: Starts with http:// or https:// and has no spaces or newlines
	if (strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) && 
		!strings.Contains(trimmed, "\n") && 
		!strings.Contains(trimmed, " ") {
		return "link"
	}

	// 2. Code Block Detection: Contains markdown code block tags
	if strings.Contains(trimmed, "```") {
		return "code"
	}

	// 3. Q&A Detection: Look for Q: and A: or User: and AI/Assistant:
	bodyLower := strings.ToLower(trimmed)
	if (strings.Contains(bodyLower, "q:") && strings.Contains(bodyLower, "a:")) ||
		(strings.Contains(bodyLower, "user:") && strings.Contains(bodyLower, "assistant:")) ||
		(strings.Contains(bodyLower, "user:") && strings.Contains(bodyLower, "ai:")) ||
		(strings.Contains(bodyLower, "human:") && strings.Contains(bodyLower, "assistant:")) {
		return "qa"
	}

	return "note"
}

// Helper to query python embeddings sidecar
func getEmbedding(text string) ([]float32, error) {
	payload := map[string]string{"text": text}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://embeddings:8000/embed", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("embedding service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return result.Embedding, nil
}

// Create a new capture (generating embedding vector on backend)
func (h *CapturesHandler) Create(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input struct {
		Title     string `json:"title"`
		Body      string `json:"body"`
		Project   string `json:"project"`
		SourceURL string `json:"source_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(input.Body) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Capture content body is required"})
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = generateAutoTitle(input.Body)
	}

	project := strings.TrimSpace(input.Project)
	if project == "" {
		project = "Inbox"
	}

	// Generate embedding vector on the backend
	embeddingText := title + "\n" + input.Body
	vector, err := getEmbedding(embeddingText)
	if err != nil {
		log.Printf("[CapturesHandler] Failed to get embedding from sidecar: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate semantic embedding"})
	}

	cType := classifyContentType(input.Body)

	now := time.Now().Unix()
	capture := db.Capture{
		ID:        uuid.New().String(),
		UserID:    userID,
		Project:   project,
		Title:     title,
		Body:      input.Body,
		SourceURL: input.SourceURL,
		Type:      cType,
		Embedding: vector,
		CreatedAt: now,
		UpdatedAt: now,
	}

	payload := actor.SaveCapturePayload{Capture: capture}
	_, err = h.gateway.Send(actor.TypeSaveCapture, payload, 5*time.Second)
	if err != nil {
		log.Printf("[CapturesHandler] SaveCapture failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save capture"})
	}

	return c.Status(201).JSON(capture)
}

// List captures (optional project filter)
func (h *CapturesHandler) List(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	projectFilter := c.Query("project")

	payload := actor.ListCapturesPayload{
		UserID:        userID,
		ProjectFilter: projectFilter,
	}
	res, err := h.gateway.Send(actor.TypeListCaptures, payload, 5*time.Second)
	if err != nil {
		log.Printf("[CapturesHandler] ListCaptures failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve captures"})
	}

	captures, ok := res.([]db.Capture)
	if !ok {
		captures = []db.Capture{}
	}

	// Sort newest first
	sort.Slice(captures, func(i, j int) bool {
		return captures[i].CreatedAt > captures[j].CreatedAt
	})

	return c.JSON(captures)
}

// Get single capture detail
func (h *CapturesHandler) Get(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	captureID := c.Params("id")

	payload := actor.GetCapturePayload{
		UserID: userID,
		ID:     captureID,
	}
	res, err := h.gateway.Send(actor.TypeGetCapture, payload, 5*time.Second)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}

	capture, ok := res.(db.Capture)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}

	return c.JSON(capture)
}

// Update an existing capture
func (h *CapturesHandler) Update(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	captureID := c.Params("id")

	// Fetch existing
	fetchRes, err := h.gateway.Send(actor.TypeGetCapture, actor.GetCapturePayload{UserID: userID, ID: captureID}, 5*time.Second)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}
	existing, ok := fetchRes.(db.Capture)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}

	var input struct {
		Title     string `json:"title"`
		Body      string `json:"body"`
		Project   string `json:"project"`
		SourceURL string `json:"source_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	textChanged := false
	if strings.TrimSpace(input.Body) != "" && input.Body != existing.Body {
		existing.Body = input.Body
		textChanged = true
	}
	
	// Update title
	if input.Title != "" && input.Title != existing.Title {
		existing.Title = strings.TrimSpace(input.Title)
		textChanged = true
	} else if textChanged {
		// Regenerate title if body updated and title not provided
		existing.Title = generateAutoTitle(existing.Body)
	}

	if input.Project != "" {
		existing.Project = strings.TrimSpace(input.Project)
	}
	
	existing.SourceURL = input.SourceURL

	if textChanged {
		embeddingText := existing.Title + "\n" + existing.Body
		vector, err := getEmbedding(embeddingText)
		if err != nil {
			log.Printf("[CapturesHandler] Failed to get embedding from sidecar: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update semantic embedding"})
		}
		existing.Embedding = vector
		existing.Type = classifyContentType(existing.Body)
	}

	existing.UpdatedAt = time.Now().Unix()

	// Save
	_, err = h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: existing}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update capture"})
	}

	return c.JSON(existing)
}

// Delete a capture
func (h *CapturesHandler) Delete(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	captureID := c.Params("id")

	payload := actor.DeleteCapturePayload{
		UserID: userID,
		ID:     captureID,
	}
	_, err := h.gateway.Send(actor.TypeDeleteCapture, payload, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete capture"})
	}

	return c.JSON(fiber.Map{"message": "Capture deleted successfully"})
}

// List user's unique projects
func (h *CapturesHandler) ListProjects(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	payload := actor.ListProjectsPayload{UserID: userID}
	res, err := h.gateway.Send(actor.TypeListProjects, payload, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve projects"})
	}

	projects, ok := res.([]string)
	if !ok {
		projects = []string{"Inbox"}
	}

	return c.JSON(projects)
}

type SearchResult struct {
	Capture    db.Capture `json:"capture"`
	Similarity float32    `json:"similarity"`
}

// Perform Semantic Vector Search
func (h *CapturesHandler) Search(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input struct {
		Query string `json:"query"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(input.Query) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Query text is required"})
	}

	// Fetch embedding from sidecar
	queryVector, err := getEmbedding(input.Query)
	if err != nil {
		log.Printf("[CapturesHandler] Failed to get embedding for query: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate search embedding"})
	}

	// Fetch all captures for user
	listPayload := actor.ListCapturesPayload{UserID: userID, ProjectFilter: ""}
	res, err := h.gateway.Send(actor.TypeListCaptures, listPayload, 5*time.Second)
	if err != nil {
		if err == actor.ErrActorUnavailable {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "database actor restarting"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list captures for semantic search"})
	}

	captures, ok := res.([]db.Capture)
	if !ok || len(captures) == 0 {
		return c.JSON([]SearchResult{})
	}

	// Compute cosine similarity in memory
	var results []SearchResult
	for _, cap := range captures {
		var similarity float32
		if len(cap.Embedding) > 0 {
			similarity = utils.CosineSimilarity(queryVector, cap.Embedding)
		} else {
			// Captures created without vector get a 0 score
			similarity = 0.0
		}

		results = append(results, SearchResult{
			Capture:    cap,
			Similarity: similarity,
		})
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return c.JSON(results)
}

package web

import (
	"bytes"
	"encoding/base64"
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

const (
	defaultSearchMinSimilarity = float32(0.70)
	defaultSearchLimit         = 20
	maxSearchLimit             = 50
)

type CapturesHandler struct {
	gateway *actor.ActorGateway
	billing *BillingHandler
}

func NewCapturesHandler(gateway *actor.ActorGateway, billing *BillingHandler) *CapturesHandler {
	return &CapturesHandler{gateway: gateway, billing: billing}
}

// Generate an automatic title from capture body text
func generateAutoTitle(body string) string {
	cleaned := strings.TrimSpace(body)
	if cleaned == "" {
		return "Untitled Capture"
	}

	// Take the first non-empty line that is NOT a markdown code block fence (e.g. ``` or ```go)
	lines := strings.Split(cleaned, "\n")
	var firstLine string
	for _, l := range lines {
		lTrimmed := strings.TrimSpace(l)
		if lTrimmed != "" && !strings.HasPrefix(lTrimmed, "```") {
			firstLine = lTrimmed
			break
		}
	}

	if firstLine == "" {
		// Fallback if all lines start with backticks
		for _, l := range lines {
			lTrimmed := strings.TrimSpace(l)
			if lTrimmed != "" {
				firstLine = strings.ReplaceAll(lTrimmed, "`", "")
				break
			}
		}
	}

	if firstLine == "" {
		firstLine = "Untitled Capture"
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

func buildEmbeddingText(title, body string, tags []string) string {
	return utils.CaptureSearchText(title, body, tags)
}

func getSuggestedTags(title, body string, existing []string) ([]string, error) {
	payload := map[string]interface{}{
		"title":         title,
		"body":          body,
		"existing_tags": existing,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tag suggestion payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://embeddings:8000/suggest-tags", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("tag suggestion service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tag suggestion service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tag suggestion response: %w", err)
	}

	return utils.NormalizeTags(result.Tags), nil
}

func resolveCaptureTags(title, body string, userTags []string) []string {
	merged := utils.MergeTags(userTags, utils.ParseHashtags(body))
	suggested, err := getSuggestedTags(title, body, merged)
	if err != nil {
		log.Printf("[CapturesHandler] Tag suggestion unavailable, using manual/hashtag tags only: %v", err)
		return merged
	}
	return utils.MergeTags(merged, suggested)
}

func tagsEqual(a, b []string) bool {
	return strings.Join(utils.NormalizeTags(a), ",") == strings.Join(utils.NormalizeTags(b), ",")
}

// Create a new capture (client-encrypted or legacy plaintext).
func (h *CapturesHandler) Create(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.billing.loadUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load user profile"})
	}
	if resp := h.billing.EnforceCaptureLimit(c, user); resp != nil {
		return resp
	}

	var input struct {
		Ciphertext string    `json:"ciphertext"`
		Embedding  []float32 `json:"embedding"`
		Title      string    `json:"title"`
		Body       string    `json:"body"`
		ProjectID  string    `json:"project_id"`
		SourceURL  string    `json:"source_url"`
		Tags       []string  `json:"tags"`
		Type       string    `json:"type"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		projectID = "inbox"
	}

	now := time.Now().Unix()

	if strings.TrimSpace(input.Ciphertext) != "" {
		ct, err := base64.StdEncoding.DecodeString(input.Ciphertext)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ciphertext encoding"})
		}
		cType := strings.TrimSpace(input.Type)
		if cType == "" {
			cType = "note"
		}
		capture := db.Capture{
			ID:         uuid.New().String(),
			UserID:     userID,
			ProjectID:  projectID,
			Ciphertext: ct,
			Type:       cType,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := prepareCaptureForStorage(&capture, input.Embedding); err != nil {
			log.Printf("[CapturesHandler] Failed to encrypt embedding: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to store embedding"})
		}
		if _, err := h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: capture}, 5*time.Second); err != nil {
			log.Printf("[CapturesHandler] SaveCapture failed: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save capture"})
		}
		return c.Status(201).JSON(toCaptureAPI(capture))
	}

	// Legacy plaintext path (pre-E2E clients / migration).
	if strings.TrimSpace(input.Body) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Capture content body or ciphertext is required"})
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = generateAutoTitle(input.Body)
	}

	tags := resolveCaptureTags(title, input.Body, utils.NormalizeTags(input.Tags))
	embeddingText := buildEmbeddingText(title, input.Body, tags)
	vector, err := getEmbedding(embeddingText)
	if err != nil {
		log.Printf("[CapturesHandler] Embedding unavailable on create, saving without vector: %v", err)
		vector = nil
	}

	cType := classifyContentType(input.Body)
	capture := db.Capture{
		ID:              uuid.New().String(),
		UserID:          userID,
		ProjectID:       projectID,
		Title:           title,
		Body:            input.Body,
		SourceURL:       input.SourceURL,
		Type:            cType,
		Tags:            tags,
		LegacyEmbedding: vector,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if _, err := h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: capture}, 5*time.Second); err != nil {
		log.Printf("[CapturesHandler] SaveCapture failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save capture"})
	}

	return c.Status(201).JSON(toCaptureAPI(capture))
}

// List captures (optional project filter)
func (h *CapturesHandler) List(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	projectFilter := c.Query("project_id")

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
	if !ok || captures == nil {
		captures = []db.Capture{}
	}

	// Sort newest first
	sort.Slice(captures, func(i, j int) bool {
		return captures[i].CreatedAt > captures[j].CreatedAt
	})

	return c.JSON(toCaptureAPIList(captures))
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

	return c.JSON(toCaptureAPI(capture))
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
		Ciphertext string    `json:"ciphertext"`
		Embedding  []float32 `json:"embedding"`
		Title      string    `json:"title"`
		Body       string    `json:"body"`
		ProjectID  string    `json:"project_id"`
		SourceURL  string    `json:"source_url"`
		Tags       *[]string `json:"tags"`
		Type       string    `json:"type"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(input.Ciphertext) != "" {
		ct, err := base64.StdEncoding.DecodeString(input.Ciphertext)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ciphertext encoding"})
		}
		existing.Ciphertext = ct
		existing.Title = ""
		existing.Body = ""
		existing.SourceURL = ""
		existing.Tags = nil
		if input.Type != "" {
			existing.Type = strings.TrimSpace(input.Type)
		}
		if input.ProjectID != "" {
			existing.ProjectID = strings.TrimSpace(input.ProjectID)
		}
		if len(input.Embedding) > 0 {
			if err := prepareCaptureForStorage(&existing, input.Embedding); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Failed to store embedding"})
			}
		}
		existing.UpdatedAt = time.Now().Unix()
		if _, err := h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: existing}, 5*time.Second); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update capture"})
		}
		return c.JSON(toCaptureAPI(existing))
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

	if input.ProjectID != "" {
		existing.ProjectID = strings.TrimSpace(input.ProjectID)
	}

	existing.SourceURL = input.SourceURL

	tagsChanged := false
	if input.Tags != nil {
		normalizedTags := utils.NormalizeTags(*input.Tags)
		tagsChanged = !tagsEqual(existing.Tags, normalizedTags)
		existing.Tags = normalizedTags
	}

	if textChanged {
		existing.Tags = resolveCaptureTags(existing.Title, existing.Body, existing.Tags)
		existing.Type = classifyContentType(existing.Body)
	}

	if textChanged || tagsChanged {
		embeddingText := buildEmbeddingText(existing.Title, existing.Body, existing.Tags)
		vector, err := getEmbedding(embeddingText)
		if err != nil {
			log.Printf("[CapturesHandler] Embedding unavailable on update, keeping prior vector: %v", err)
		} else {
			existing.LegacyEmbedding = vector
		}
	}

	existing.UpdatedAt = time.Now().Unix()

	// Save
	_, err = h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: existing}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update capture"})
	}

	return c.JSON(toCaptureAPI(existing))
}

// Delete a capture (Soft Delete if active, Hard Delete if already deleted)
func (h *CapturesHandler) Delete(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	captureID := c.Params("id")

	// 1. Fetch the capture to check current state
	fetchRes, err := h.gateway.Send(actor.TypeGetCapture, actor.GetCapturePayload{UserID: userID, ID: captureID}, 5*time.Second)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}
	capture, ok := fetchRes.(db.Capture)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}

	// 2. If it is already in Trash (DeletedAt > 0), perform Hard Delete
	if capture.DeletedAt > 0 {
		payload := actor.DeleteCapturePayload{
			UserID: userID,
			ID:     captureID,
		}
		_, err := h.gateway.Send(actor.TypeDeleteCapture, payload, 5*time.Second)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to permanently delete capture"})
		}
		return c.JSON(fiber.Map{"message": "Capture permanently deleted successfully"})
	}

	// 3. Otherwise, set DeletedAt and perform Soft Delete
	capture.DeletedAt = time.Now().Unix()
	capture.UpdatedAt = time.Now().Unix()
	_, err = h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: capture}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to move capture to Trash"})
	}

	return c.JSON(fiber.Map{"message": "Capture moved to Trash successfully"})
}

// Restore a soft-deleted capture
func (h *CapturesHandler) Restore(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	captureID := c.Params("id")

	// Fetch existing capture
	fetchRes, err := h.gateway.Send(actor.TypeGetCapture, actor.GetCapturePayload{UserID: userID, ID: captureID}, 5*time.Second)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}
	capture, ok := fetchRes.(db.Capture)
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "Capture not found"})
	}

	capture.DeletedAt = 0
	capture.UpdatedAt = time.Now().Unix()
	_, err = h.gateway.Send(actor.TypeSaveCapture, actor.SaveCapturePayload{Capture: capture}, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to restore capture"})
	}

	return c.JSON(toCaptureAPI(capture))
}

// EmptyTrash deletes all soft-deleted captures permanently
func (h *CapturesHandler) EmptyTrash(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Fetch all captures for the user with projectFilter == "Trash" (returns soft-deleted captures)
	payload := actor.ListCapturesPayload{
		UserID:        userID,
		ProjectFilter: "Trash",
	}
	res, err := h.gateway.Send(actor.TypeListCaptures, payload, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list trash captures"})
	}
	captures, ok := res.([]db.Capture)
	if !ok {
		captures = []db.Capture{}
	}

	deletedCount := 0
	for _, cap := range captures {
		delPayload := actor.DeleteCapturePayload{
			UserID: userID,
			ID:     cap.ID,
		}
		_, err = h.gateway.Send(actor.TypeDeleteCapture, delPayload, 5*time.Second)
		if err != nil {
			log.Printf("[CapturesHandler] EmptyTrash: failed to delete capture %s: %v", cap.ID, err)
		} else {
			deletedCount++
		}
	}

	return c.JSON(fiber.Map{"message": "Trash emptied successfully", "count": deletedCount})
}

type projectAPI struct {
	ID         string `json:"id"`
	Ciphertext string `json:"ciphertext"`
	CreatedAt  int64  `json:"created_at"`
}

// List user's encrypted projects (the "inbox" default is a frontend-only sentinel,
// never stored here).
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

	projects, ok := res.([]db.Project)
	if !ok {
		projects = []db.Project{}
	}

	out := make([]projectAPI, 0, len(projects))
	for _, p := range projects {
		out = append(out, projectAPI{
			ID:         p.ID,
			Ciphertext: base64.StdEncoding.EncodeToString(p.Ciphertext),
			CreatedAt:  p.CreatedAt,
		})
	}

	return c.JSON(out)
}

// CreateProject saves a client-encrypted project name under a client-generated ID.
func (h *CapturesHandler) CreateProject(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input struct {
		ID         string `json:"id"`
		Ciphertext string `json:"ciphertext"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	id := strings.TrimSpace(input.ID)
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Project id is required"})
	}

	ct, err := base64.StdEncoding.DecodeString(input.Ciphertext)
	if err != nil || len(ct) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ciphertext encoding"})
	}

	project := db.Project{
		ID:         id,
		UserID:     userID,
		Ciphertext: ct,
		CreatedAt:  time.Now().Unix(),
	}
	if _, err := h.gateway.Send(actor.TypeSaveProject, actor.SaveProjectPayload{Project: project}, 5*time.Second); err != nil {
		log.Printf("[CapturesHandler] SaveProject failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save project"})
	}

	return c.Status(201).JSON(projectAPI{
		ID:         project.ID,
		Ciphertext: input.Ciphertext,
		CreatedAt:  project.CreatedAt,
	})
}

// ListTags returns unique tags across the user's captures (for autocomplete).
func (h *CapturesHandler) ListTags(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	payload := actor.ListCapturesPayload{UserID: userID}
	res, err := h.gateway.Send(actor.TypeListCaptures, payload, 5*time.Second)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve tags"})
	}

	captures, ok := res.([]db.Capture)
	if !ok || captures == nil {
		return c.JSON([]string{})
	}

	seen := make(map[string]struct{})
	for _, cap := range captures {
		if cap.IsEncrypted() {
			continue
		}
		for _, tag := range cap.Tags {
			n := utils.NormalizeTag(tag)
			if n != "" {
				seen[n] = struct{}{}
			}
		}
	}

	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return c.JSON(tags)
}

type SearchResult struct {
	Capture    captureAPI `json:"capture"`
	Similarity float32    `json:"similarity"`
}

type searchRequest struct {
	Query           string    `json:"query"`
	QueryEmbedding  []float32 `json:"query_embedding"`
	ProjectID       string    `json:"project_id"`
	MinSimilarity   *float32  `json:"min_similarity"`
	Limit           int       `json:"limit"`
	ExcludeID       string    `json:"exclude_id"`
}

func rankCapturesBySimilarity(query string, queryVector []float32, captures []db.Capture, opts searchRequest) []SearchResult {
	minSim := defaultSearchMinSimilarity
	if opts.MinSimilarity != nil {
		minSim = *opts.MinSimilarity
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	var results []SearchResult
	for _, cap := range captures {
		if opts.ExcludeID != "" && cap.ID == opts.ExcludeID {
			continue
		}

		capVector := captureSearchVector(cap)
		if len(queryVector) == 0 || len(capVector) == 0 {
			continue
		}
		similarity := utils.CosineSimilarity(queryVector, capVector)
		if similarity < minSim {
			continue
		}

		results = append(results, SearchResult{
			Capture:    toCaptureAPI(cap),
			Similarity: similarity,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}
	if results == nil {
		results = []SearchResult{}
	}
	return results
}

// Perform Semantic Vector Search
func (h *CapturesHandler) Search(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input searchRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	query := strings.TrimSpace(input.Query)
	queryVector := input.QueryEmbedding

	if len(queryVector) == 0 {
		if query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "query_embedding or query text is required"})
		}
		var err error
		queryVector, err = getEmbedding(query)
		if err != nil {
			log.Printf("[CapturesHandler] Embedding unavailable for search, using text similarity: %v", err)
			queryVector = nil
		}
	}

	projectFilter := strings.TrimSpace(input.ProjectID)
	listPayload := actor.ListCapturesPayload{UserID: userID, ProjectFilter: projectFilter}
	res, err := h.gateway.Send(actor.TypeListCaptures, listPayload, 5*time.Second)
	if err != nil {
		if err == actor.ErrActorUnavailable {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "database actor restarting"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list captures for semantic search"})
	}

	captures, ok := res.([]db.Capture)
	if !ok || captures == nil {
		captures = []db.Capture{}
	}

	return c.JSON(rankCapturesBySimilarity(query, queryVector, captures, input))
}

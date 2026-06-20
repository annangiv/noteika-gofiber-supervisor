package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/bits"
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
	defaultSearchLimit = 20
	maxSearchLimit     = 50
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
		Ciphertext      string   `json:"ciphertext"`
		Fingerprint     string   `json:"fingerprint"`
		EncryptedVector string   `json:"encrypted_vector"`
		Title           string   `json:"title"`
		Body            string   `json:"body"`
		ProjectID       string   `json:"project_id"`
		SourceURL       string   `json:"source_url"`
		Tags            []string `json:"tags"`
		Type            string   `json:"type"`
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
		if strings.TrimSpace(input.Fingerprint) != "" {
			fp, err := base64.StdEncoding.DecodeString(input.Fingerprint)
			if err != nil || len(fp) != 32 {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid fingerprint: must be base64-encoded 32 bytes"})
			}
			capture.Fingerprint = fp
		}
		if strings.TrimSpace(input.EncryptedVector) != "" {
			ev, err := base64.StdEncoding.DecodeString(input.EncryptedVector)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid encrypted_vector encoding"})
			}
			capture.EncryptedVector = ev
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

	cType := classifyContentType(input.Body)
	capture := db.Capture{
		ID:        uuid.New().String(),
		UserID:    userID,
		ProjectID: projectID,
		Title:     title,
		Body:      input.Body,
		SourceURL: input.SourceURL,
		Type:      cType,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
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
		Ciphertext      string    `json:"ciphertext"`
		Fingerprint     string    `json:"fingerprint"`
		EncryptedVector string    `json:"encrypted_vector"`
		Title           string    `json:"title"`
		Body            string    `json:"body"`
		ProjectID       string    `json:"project_id"`
		SourceURL       string    `json:"source_url"`
		Tags            *[]string `json:"tags"`
		Type            string    `json:"type"`
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
		if strings.TrimSpace(input.Fingerprint) != "" {
			fp, err := base64.StdEncoding.DecodeString(input.Fingerprint)
			if err != nil || len(fp) != 32 {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid fingerprint: must be base64-encoded 32 bytes"})
			}
			existing.Fingerprint = fp
		}
		if strings.TrimSpace(input.EncryptedVector) != "" {
			ev, err := base64.StdEncoding.DecodeString(input.EncryptedVector)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid encrypted_vector encoding"})
			}
			existing.EncryptedVector = ev
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

	if input.Tags != nil {
		existing.Tags = utils.NormalizeTags(*input.Tags)
	}

	if textChanged {
		existing.Tags = resolveCaptureTags(existing.Title, existing.Body, existing.Tags)
		existing.Type = classifyContentType(existing.Body)
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
	QueryFingerprint string `json:"query_fingerprint"`
	ProjectID        string `json:"project_id"`
	Limit            int    `json:"limit"`
	ExcludeID        string `json:"exclude_id"`
}

// hammingDistance counts differing bits between two equal-length byte slices.
func hammingDistance(a, b []byte) int {
	dist := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		dist += bits.OnesCount8(a[i] ^ b[i])
	}
	return dist
}

// rankCapturesByFingerprint ranks captures by Hamming distance between their
// stored fingerprint and the query fingerprint, converting distance to a
// cosine-like score via the SimHash plug-in estimator cos(pi * hamming / 256).
// This is an approximate pre-filter only — the client re-ranks exactly on the
// decrypted candidates using the real embeddings.
func rankCapturesByFingerprint(queryFP []byte, captures []db.Capture, opts searchRequest) []SearchResult {
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
		if len(queryFP) != 32 || len(cap.Fingerprint) != 32 {
			continue
		}

		dist := hammingDistance(queryFP, cap.Fingerprint)
		similarity := float32(math.Cos(math.Pi * float64(dist) / 256))

		capAPI := toCaptureAPI(cap)
		if len(cap.EncryptedVector) > 0 {
			capAPI.EncryptedVector = base64.StdEncoding.EncodeToString(cap.EncryptedVector)
		}
		results = append(results, SearchResult{
			Capture:    capAPI,
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

// Search performs an approximate fingerprint-based candidate search. The
// server never sees a real embedding — only a one-way-ish binarized
// fingerprint — so this can only narrow candidates, not exactly rank them;
// the client decrypts the returned candidates and re-ranks exactly.
func (h *CapturesHandler) Search(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var input searchRequest
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	queryFP, err := base64.StdEncoding.DecodeString(input.QueryFingerprint)
	if err != nil || len(queryFP) != 32 {
		return c.Status(400).JSON(fiber.Map{"error": "query_fingerprint must be base64-encoded 32 bytes"})
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

	return c.JSON(rankCapturesByFingerprint(queryFP, captures, input))
}

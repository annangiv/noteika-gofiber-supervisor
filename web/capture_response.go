package web

import (
	"encoding/base64"

	"my-app/db"
)

type captureAPI struct {
	ID         string   `json:"id"`
	UserID     string   `json:"user_id,omitempty"`
	ProjectID  string   `json:"project_id"`
	Type       string   `json:"type"`
	Ciphertext string   `json:"ciphertext,omitempty"`
	Title      string   `json:"title,omitempty"`
	Body       string   `json:"body,omitempty"`
	SourceURL  string   `json:"source_url,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	CreatedAt  int64    `json:"created_at"`
	UpdatedAt  int64    `json:"updated_at"`
	DeletedAt  int64    `json:"deleted_at"`
}

func toCaptureAPI(c db.Capture) captureAPI {
	out := captureAPI{
		ID:        c.ID,
		ProjectID: c.ProjectID,
		Type:      c.Type,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,
	}
	if c.IsEncrypted() {
		out.Ciphertext = base64.StdEncoding.EncodeToString(c.Ciphertext)
		return out
	}
	out.Title = c.Title
	out.Body = c.Body
	out.SourceURL = c.SourceURL
	out.Tags = c.Tags
	return out
}

func toCaptureAPIList(captures []db.Capture) []captureAPI {
	out := make([]captureAPI, 0, len(captures))
	for _, c := range captures {
		out = append(out, toCaptureAPI(c))
	}
	return out
}

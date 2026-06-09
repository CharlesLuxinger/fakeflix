package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/database"
	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

//nolint:gocyclo,funlen,paralleltest // integration test uses shared DB; parallel runs would interfere
func TestVideoUploadPersistsToPostgres(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, closeDB, err := database.Open(t.Context(), dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}

	t.Cleanup(func() {
		// Clean up test data
		db.Exec("DELETE FROM videos")

		if err := closeDB(); err != nil {
			t.Logf("close database: %v", err)
		}
	})

	uploadsDir := t.TempDir()
	videoSvc := service.NewVideoService(uploadsDir)
	videoRepo := repository.NewVideoRepository(db)
	h := handler.NewVideo(videoSvc, videoRepo, uploadsDir)

	// Build multipart request
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("title", "Integration Test Video"); err != nil {
		t.Fatalf("write title: %v", err)
	}

	if err := writer.WriteField("description", "Integration Test Description"); err != nil {
		t.Fatalf("write description: %v", err)
	}

	// Add video file with explicit Content-Type header
	videoHeader := make(textproto.MIMEHeader)
	videoHeader.Set("Content-Disposition", `form-data; name="video"; filename="test.mp4"`)
	videoHeader.Set("Content-Type", "video/mp4")
	videoPart, err := writer.CreatePart(videoHeader)
	if err != nil { //nolint:wsl // gofumpt forbids blank line before if err; wsl wants one
		t.Fatalf("create video form part: %v", err)
	}

	if _, err = videoPart.Write(bytes.Repeat([]byte("x"), 2048)); err != nil {
		t.Fatalf("write video content: %v", err)
	}

	// Add thumbnail file with explicit Content-Type header
	thumbHeader := make(textproto.MIMEHeader)
	thumbHeader.Set("Content-Disposition", `form-data; name="thumbnail"; filename="test.jpg"`)
	thumbHeader.Set("Content-Type", "image/jpeg")
	thumbPart, err := writer.CreatePart(thumbHeader)
	if err != nil { //nolint:wsl // gofumpt forbids blank line before if err; wsl wants one
		t.Fatalf("create thumbnail form part: %v", err)
	}

	if _, err = thumbPart.Write([]byte("fake jpeg content")); err != nil {
		t.Fatalf("write thumbnail content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/videos", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify response JSON
	var respBody struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		URL          string `json:"url"`
		ThumbnailURL string `json:"thumbnailUrl"`
		SizeInKB     int    `json:"sizeInKb"`
		Duration     int    `json:"duration"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if respBody.ID == "" {
		t.Error("id is empty")
	}

	if respBody.Title != "Integration Test Video" {
		t.Errorf("title = %q, want %q", respBody.Title, "Integration Test Video")
	}

	if respBody.Description != "Integration Test Description" {
		t.Errorf("description = %q, want %q", respBody.Description, "Integration Test Description")
	}

	if respBody.Duration != 100 {
		t.Errorf("duration = %d, want 100", respBody.Duration)
	}

	if respBody.URL == "" || !strings.HasPrefix(respBody.URL, "uploads/") {
		t.Errorf("url = %q, want uploads/ prefix", respBody.URL)
	}

	if respBody.ThumbnailURL == "" || !strings.HasPrefix(respBody.ThumbnailURL, "uploads/") {
		t.Errorf("thumbnailUrl = %q, want uploads/ prefix", respBody.ThumbnailURL)
	}

	if respBody.SizeInKB <= 0 {
		t.Errorf("sizeInKb = %d, want > 0", respBody.SizeInKB)
	}

	// Query PostgreSQL directly to confirm persistence
	var count int
	if err := db.Raw("SELECT COUNT(*) FROM videos WHERE id = ?", respBody.ID).Scan(&count).Error; err != nil {
		t.Fatalf("query videos: %v", err)
	}

	if count != 1 {
		t.Errorf("videos count = %d, want 1", count)
	}

	// Query for title and duration to verify
	type dbVideo struct {
		Title       string
		Description string
		Duration    int
	}

	var dbRecord dbVideo
	if err := db.Raw("SELECT title, description, duration FROM videos WHERE id = ? ORDER BY created_at DESC LIMIT 1", respBody.ID).Scan(&dbRecord).Error; err != nil {
		t.Fatalf("query video record: %v", err)
	}

	if dbRecord.Title != "Integration Test Video" {
		t.Errorf("db title = %q, want %q", dbRecord.Title, "Integration Test Video")
	}

	if dbRecord.Description != "Integration Test Description" {
		t.Errorf("db description = %q, want %q", dbRecord.Description, "Integration Test Description")
	}

	if dbRecord.Duration != 100 {
		t.Errorf("db duration = %d, want 100", dbRecord.Duration)
	}
}

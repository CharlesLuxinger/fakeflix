package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/database"
	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
	"github.com/google/uuid"
)

//nolint:funlen,gocyclo,paralleltest // integration test uses shared DB
func TestVideoUploadThenStreamIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, closeDB, err := database.Open(t.Context(), dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM videos")

		if err := closeDB(); err != nil {
			t.Logf("close database: %v", err)
		}
	})

	uploadsDir := t.TempDir()
	videoSvc := service.NewVideoService(uploadsDir)
	videoRepo := repository.NewVideoRepository(db)
	uploadHandler := handler.NewVideo(videoSvc, videoRepo, uploadsDir)

	// Upload a video
	videoContent := bytes.Repeat([]byte("A"), 4096)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("title", "Stream Integration Video"); err != nil {
		t.Fatalf("write title: %v", err)
	}

	if err := writer.WriteField("description", "Stream Integration Description"); err != nil {
		t.Fatalf("write description: %v", err)
	}

	videoHeader := make(textproto.MIMEHeader)
	videoHeader.Set("Content-Disposition", `form-data; name="video"; filename="stream-test.mp4"`)
	videoHeader.Set("Content-Type", "video/mp4")

	videoPart, err := writer.CreatePart(videoHeader)
	if err != nil {
		t.Fatalf("create video form part: %v", err)
	}

	if _, err = videoPart.Write(videoContent); err != nil {
		t.Fatalf("write video content: %v", err)
	}

	thumbHeader := make(textproto.MIMEHeader)
	thumbHeader.Set("Content-Disposition", `form-data; name="thumbnail"; filename="stream-test.jpg"`)
	thumbHeader.Set("Content-Type", "image/jpeg")

	thumbPart, err := writer.CreatePart(thumbHeader)
	if err != nil {
		t.Fatalf("create thumbnail form part: %v", err)
	}

	if _, err = thumbPart.Write([]byte("fake jpeg content")); err != nil {
		t.Fatalf("write thumbnail content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/videos", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	uploadRec := httptest.NewRecorder()
	uploadHandler.ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d; body = %s", uploadRec.Code, http.StatusCreated, uploadRec.Body.String())
	}

	var uploadResp struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}

	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("unmarshal upload response: %v", err)
	}

	if uploadResp.ID == "" {
		t.Fatal("upload response has empty ID")
	}

	// Stream the uploaded video with Range header
	streamHandler := handler.NewVideoStream(videoRepo, uploadsDir)

	t.Run("stream uploaded video with range", func(t *testing.T) {
		streamReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream/"+uploadResp.ID, nil)
		streamReq.SetPathValue("videoId", uploadResp.ID)
		streamReq.Header.Set("Range", "bytes=0-3")

		streamRec := httptest.NewRecorder()
		streamHandler.ServeHTTP(streamRec, streamReq)

		if streamRec.Code != http.StatusPartialContent {
			t.Errorf("stream status = %d, want %d; body = %q", streamRec.Code, http.StatusPartialContent, streamRec.Body.String())
		}

		if streamRec.Header().Get("Content-Type") != "video/mp4" {
			t.Errorf("Content-Type = %q, want video/mp4", streamRec.Header().Get("Content-Type"))
		}

		if streamRec.Header().Get("Accept-Ranges") != "bytes" {
			t.Errorf("Accept-Ranges = %q, want bytes", streamRec.Header().Get("Accept-Ranges"))
		}

		wantBody := videoContent[:4]
		if streamRec.Body.String() != string(wantBody) {
			t.Errorf("body = %q, want %q", streamRec.Body.String(), string(wantBody))
		}
	})

	t.Run("stream uploaded video full GET", func(t *testing.T) {
		streamReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream/"+uploadResp.ID, nil)
		streamReq.SetPathValue("videoId", uploadResp.ID)

		streamRec := httptest.NewRecorder()
		streamHandler.ServeHTTP(streamRec, streamReq)

		if streamRec.Code != http.StatusOK {
			t.Errorf("stream status = %d, want %d; body = %q", streamRec.Code, http.StatusOK, streamRec.Body.String())
		}

		if streamRec.Body.Len() != len(videoContent) {
			t.Errorf("body length = %d, want %d", streamRec.Body.Len(), len(videoContent))
		}
	})

	t.Run("unknown stream ID returns 404", func(t *testing.T) {
		unknownID := uuid.New().String()
		streamReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream/"+unknownID, nil)
		streamReq.SetPathValue("videoId", unknownID)

		streamRec := httptest.NewRecorder()
		streamHandler.ServeHTTP(streamRec, streamReq)

		if streamRec.Code != http.StatusNotFound {
			t.Errorf("stream status = %d, want %d; body = %q", streamRec.Code, http.StatusNotFound, streamRec.Body.String())
		}
	})
}

//nolint:paralleltest // uses shared DB
func TestVideoStream_UnknownID_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, closeDB, err := database.Open(t.Context(), dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}

	t.Cleanup(func() {
		if err := closeDB(); err != nil {
			t.Logf("close database: %v", err)
		}
	})

	videoRepo := repository.NewVideoRepository(db)
	streamHandler := handler.NewVideoStream(videoRepo, t.TempDir())
	unknownID := uuid.New().String()

	streamReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream/"+unknownID, nil)
	streamReq.SetPathValue("videoId", unknownID)

	streamRec := httptest.NewRecorder()
	streamHandler.ServeHTTP(streamRec, streamReq)

	if streamRec.Code != http.StatusNotFound {
		t.Errorf("stream status = %d, want %d; body = %q", streamRec.Code, http.StatusNotFound, streamRec.Body.String())
	}
}

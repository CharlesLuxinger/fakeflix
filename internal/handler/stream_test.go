package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
)

type mockVideoFinder struct {
	video *model.Video
	err   error
}

func (m *mockVideoFinder) FindByID(_ context.Context, _ string) (*model.Video, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.video, nil
}

//nolint:funlen,gocyclo // table-driven test with multiple cases
func TestVideoStream_ServeHTTP(t *testing.T) {
	t.Parallel()

	videoContent := []byte("fake mp4 video content for testing")
	videoFilename := "test-video.mp4"

	tests := []struct {
		name        string
		method      string
		videoID     string
		videoURL    string
		repoErr     error
		createFile  bool
		rangeHeader string
		wantStatus  int
		wantHeaders map[string]string
		wantBodyLen int
		wantBody    []byte
	}{
		{
			name:        "full GET returns 200 and full body",
			method:      http.MethodGet,
			videoID:     "existing-video-id",
			videoURL:    videoFilename,
			createFile:  true,
			wantStatus:  http.StatusOK,
			wantHeaders: map[string]string{"Content-Type": "video/mp4"},
			wantBodyLen: len(videoContent),
			wantBody:    videoContent,
		},
		{
			name:        "range request returns 206 partial content",
			method:      http.MethodGet,
			videoID:     "range-video-id",
			videoURL:    videoFilename,
			createFile:  true,
			rangeHeader: "bytes=0-3",
			wantStatus:  http.StatusPartialContent,
			wantHeaders: map[string]string{
				"Content-Type":   "video/mp4",
				"Content-Range":  "bytes 0-3/" + intToStr(len(videoContent)),
				"Accept-Ranges":  "bytes",
				"Content-Length": "4",
			},
			wantBodyLen: 4,
			wantBody:    videoContent[:4],
		},
		{
			name:       "unknown video returns 404",
			method:     http.MethodGet,
			videoID:    "unknown-video-id",
			repoErr:    repository.ErrVideoNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing video file returns 404",
			method:     http.MethodGet,
			videoID:    "missing-file-video",
			videoURL:   "nonexistent.mp4",
			createFile: false,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unsafe path traversal returns 404",
			method:     http.MethodGet,
			videoID:    "unsafe-video",
			videoURL:   "../../etc/passwd",
			createFile: false,
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "invalid range returns 416",
			method:      http.MethodGet,
			videoID:     "invalid-range-video",
			videoURL:    videoFilename,
			createFile:  true,
			rangeHeader: "bytes=9999-99999",
			wantStatus:  http.StatusRequestedRangeNotSatisfiable,
		},
		{
			name:       "non-GET method returns 405",
			method:     http.MethodPost,
			videoID:    "any-video",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "blank video ID returns 404",
			method:     http.MethodGet,
			videoID:    "",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uploadsDir := t.TempDir()

			videoURL := tt.videoURL
			if tt.createFile && videoURL != "" {
				cleanName := filepath.Base(videoURL)
				path := filepath.Join(uploadsDir, cleanName)

				if err := os.WriteFile(path, videoContent, 0o600); err != nil {
					t.Fatalf("write test file: %v", err)
				}

				videoURL = "uploads/" + cleanName
			}

			mock := &mockVideoFinder{}

			if videoURL != "" || tt.repoErr != nil {
				if tt.repoErr != nil {
					mock.err = tt.repoErr
				} else {
					mock.video = &model.Video{
						ID:  tt.videoID,
						URL: videoURL,
					}
				}
			}

			streamHandler := handler.NewVideoStream(mock, uploadsDir)
			r := httptest.NewRequestWithContext(t.Context(), tt.method, "/stream/"+tt.videoID, nil)
			r.SetPathValue("videoId", tt.videoID)

			if tt.rangeHeader != "" {
				r.Header.Set("Range", tt.rangeHeader)
			}

			w := httptest.NewRecorder()
			streamHandler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			for key, want := range tt.wantHeaders {
				if got := w.Header().Get(key); got != want {
					t.Errorf("header %q = %q, want %q", key, got, want)
				}
			}

			if tt.wantBodyLen > 0 && w.Body.Len() != tt.wantBodyLen {
				t.Errorf("body length = %d, want %d", w.Body.Len(), tt.wantBodyLen)
			}

			if tt.wantBody != nil && w.Body.String() != string(tt.wantBody) {
				t.Errorf("body = %q, want %q", w.Body.String(), string(tt.wantBody))
			}
		})
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[i:])
}

// TestVideoStream_RepoError ensures non-ErrVideoNotFound errors map to 500.
func TestVideoStream_RepoError(t *testing.T) {
	t.Parallel()

	mock := &mockVideoFinder{err: errors.New("database connection refused")}
	streamHandler := handler.NewVideoStream(mock, t.TempDir())
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream/any", nil)
	r.SetPathValue("videoId", "any")

	w := httptest.NewRecorder()

	streamHandler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

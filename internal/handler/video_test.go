package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

const (
	mimeTypeMP4  = "video/mp4"
	mimeTypeJPEG = "image/jpeg"
)

type mockSaveCall struct {
	fh          *multipart.FileHeader
	allowedMIME string
}

type mockSaveReturn struct {
	filename string
	err      error
}

type mockVideoSaver struct {
	calls      []mockSaveCall
	returns    []mockSaveReturn
	callIdx    int
	uploadsDir string
}

type mockVideoRepo struct {
	videos []*model.Video
	err    error
}

func (m *mockVideoRepo) Create(_ context.Context, video *model.Video) error {
	if m.err != nil {
		return m.err
	}

	m.videos = append(m.videos, video)

	return nil
}

func (m *mockVideoSaver) SaveFile(
	fh *multipart.FileHeader,
	allowedMIME string,
) (string, error) {
	m.calls = append(m.calls, mockSaveCall{
		fh:          fh,
		allowedMIME: allowedMIME,
	})

	if m.callIdx >= len(m.returns) {
		return "", errors.New("unexpected save call")
	}

	ret := m.returns[m.callIdx]
	m.callIdx++

	if m.uploadsDir != "" && ret.filename != "" && ret.err == nil {
		path := filepath.Join(m.uploadsDir, ret.filename)
		if err := os.WriteFile(path, []byte("video"), 0o600); err != nil {
			return "", err
		}
	}

	return ret.filename, ret.err
}

//nolint:funlen // table-driven test with many cases, benefits from readability over splitting
func TestVideo_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		method          string
		includeVideo    bool
		includeThumb    bool
		includeTitle    bool
		includeDesc     bool
		invalidBody     bool
		repoErr         error
		returns         []mockSaveReturn
		useUploadsDir   bool
		wantStatus      int
		wantBody        string
		wantJSON        bool
		wantAllowedMIME []string
		wantCleanup     []string
	}{
		{
			name:          "valid mp4 and jpeg",
			method:        http.MethodPost,
			includeVideo:  true,
			includeThumb:  true,
			includeTitle:  true,
			includeDesc:   true,
			useUploadsDir: true,
			returns: []mockSaveReturn{
				{filename: "video.mp4"},
				{filename: "thumbnail.jpg"},
			},
			wantStatus:      http.StatusCreated,
			wantJSON:        true,
			wantAllowedMIME: []string{mimeTypeMP4, mimeTypeJPEG},
		},
		{
			name:         "missing title",
			method:       http.MethodPost,
			includeVideo: true,
			includeThumb: true,
			includeDesc:  true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing or blank title\n",
		},
		{
			name:         "missing description",
			method:       http.MethodPost,
			includeVideo: true,
			includeThumb: true,
			includeTitle: true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing or blank description\n",
		},
		{
			name:         "missing video field",
			method:       http.MethodPost,
			includeThumb: true,
			includeTitle: true,
			includeDesc:  true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing video file\n",
		},
		{
			name:         "missing thumbnail field",
			method:       http.MethodPost,
			includeVideo: true,
			includeTitle: true,
			includeDesc:  true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing thumbnail file\n",
		},
		{
			name:         "wrong video MIME",
			method:       http.MethodPost,
			includeVideo: true,
			includeThumb: true,
			includeTitle: true,
			includeDesc:  true,
			returns: []mockSaveReturn{
				{err: &service.ErrInvalidMIMEType{Got: mimeTypeJPEG}},
			},
			wantStatus: http.StatusBadRequest,
			wantBody: (&service.ErrInvalidMIMEType{
				Got: mimeTypeJPEG,
			}).Error() + "\n",
			wantAllowedMIME: []string{mimeTypeMP4},
		},
		{
			name:          "wrong thumbnail MIME — cleanup verified",
			method:        http.MethodPost,
			includeVideo:  true,
			includeThumb:  true,
			includeTitle:  true,
			includeDesc:   true,
			useUploadsDir: true,
			returns: []mockSaveReturn{
				{filename: "saved-video.mp4"},
				{err: &service.ErrInvalidMIMEType{}},
			},
			wantStatus:      http.StatusBadRequest,
			wantBody:        (&service.ErrInvalidMIMEType{}).Error() + "\n",
			wantAllowedMIME: []string{mimeTypeMP4, mimeTypeJPEG},
			wantCleanup:     []string{"saved-video.mp4"},
		},
		{
			name:          "db failure",
			method:        http.MethodPost,
			includeVideo:  true,
			includeThumb:  true,
			includeTitle:  true,
			includeDesc:   true,
			useUploadsDir: true,
			repoErr:       errors.New("db error"),
			returns: []mockSaveReturn{
				{filename: "db-video.mp4"},
				{filename: "db-thumbnail.jpg"},
			},
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "failed to save video record\n",
			wantAllowedMIME: []string{mimeTypeMP4, mimeTypeJPEG},
			wantCleanup:     []string{"db-video.mp4", "db-thumbnail.jpg"},
		},
		{
			name:       "non-POST method",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   http.StatusText(http.StatusMethodNotAllowed) + "\n",
		},
		{
			name:        "invalid multipart body",
			method:      http.MethodPost,
			invalidBody: true,
			wantStatus:  http.StatusBadRequest,
			wantBody:    "invalid multipart body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uploadsDir := ""
			if tt.useUploadsDir {
				uploadsDir = t.TempDir()
			}

			saver := &mockVideoSaver{
				returns:    tt.returns,
				uploadsDir: uploadsDir,
			}
			repo := &mockVideoRepo{err: tt.repoErr}
			h := handler.NewVideo(saver, repo, uploadsDir)
			r := newVideoRequest(t, tt.method, requestOptions{
				includeVideo:       tt.includeVideo,
				includeThumb:       tt.includeThumb,
				includeTitle:       tt.includeTitle,
				includeDescription: tt.includeDesc,
				invalidBody:        tt.invalidBody,
			})
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}

			if tt.wantJSON {
				assertVideoJSON(t, w.Body.Bytes())
			}

			assertSaveCalls(t, saver.calls, tt.wantAllowedMIME)

			for _, filename := range tt.wantCleanup {
				path := filepath.Join(uploadsDir, filename)
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("cleanup stat error for %q = %v, want not exists", filename, err)
				}
			}
		})
	}
}

type requestOptions struct {
	includeVideo       bool
	includeThumb       bool
	includeTitle       bool
	includeDescription bool
	invalidBody        bool
}

func newVideoRequest(
	t *testing.T,
	method string,
	options requestOptions,
) *http.Request {
	t.Helper()

	if options.invalidBody {
		r := httptest.NewRequestWithContext(t.Context(), method, "/videos", bytes.NewBufferString("bad"))
		r.Header.Set("Content-Type", "text/plain")

		return r
	}

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	if options.includeTitle {
		writeField(t, writer, "title", "Test video")
	}

	if options.includeDescription {
		writeField(t, writer, "description", "Test description")
	}

	if options.includeVideo {
		writeFormFile(t, writer, "video", "video.mp4", "video data")
	}

	if options.includeThumb {
		writeFormFile(t, writer, "thumbnail", "thumbnail.jpg", "thumb data")
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequestWithContext(t.Context(), method, "/videos", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())

	return r
}

func writeField(t *testing.T, writer *multipart.Writer, field, value string) {
	t.Helper()

	if err := writer.WriteField(field, value); err != nil {
		t.Fatalf("write field %q: %v", field, err)
	}
}

func writeFormFile(
	t *testing.T,
	writer *multipart.Writer,
	field string,
	filename string,
	content string,
) {
	t.Helper()

	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file %q: %v", field, err)
	}

	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write form file %q: %v", field, err)
	}
}

func assertVideoJSON(t *testing.T, body []byte) {
	t.Helper()

	var video model.Video
	if err := json.Unmarshal(body, &video); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if video.ID == "" {
		t.Error("id is empty")
	}

	if video.Duration != 100 {
		t.Errorf("duration = %d, want 100", video.Duration)
	}

	if video.Title != "Test video" {
		t.Errorf("title = %q, want %q", video.Title, "Test video")
	}

	if video.Description != "Test description" {
		t.Errorf("description = %q, want %q", video.Description, "Test description")
	}

	if !strings.HasPrefix(video.URL, "uploads/") {
		t.Errorf("url = %q, want uploads/ prefix", video.URL)
	}

	if !strings.HasPrefix(video.ThumbnailURL, "uploads/") {
		t.Errorf("thumbnailUrl = %q, want uploads/ prefix", video.ThumbnailURL)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal raw response body: %v", err)
	}

	var sizeInKB int
	if err := json.Unmarshal(raw["sizeInKb"], &sizeInKB); err != nil {
		t.Fatalf("sizeInKb is not an integer: %v", err)
	}
}

func assertSaveCalls(
	t *testing.T,
	calls []mockSaveCall,
	wantAllowedMIME []string,
) {
	t.Helper()

	if len(calls) != len(wantAllowedMIME) {
		t.Errorf("save calls = %d, want %d", len(calls), len(wantAllowedMIME))

		return
	}

	for i, call := range calls {
		if call.fh == nil {
			t.Errorf("save call %d file header is nil", i)
		}

		if call.allowedMIME != wantAllowedMIME[i] {
			t.Errorf(
				"save call %d allowed MIME = %q, want %q",
				i,
				call.allowedMIME,
				wantAllowedMIME[i],
			)
		}
	}
}

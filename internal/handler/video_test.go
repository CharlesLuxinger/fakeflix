package handler_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
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
		invalidBody     bool
		returns         []mockSaveReturn
		useUploadsDir   bool
		wantStatus      int
		wantBody        string
		wantAllowedMIME []string
		wantCleanup     bool
	}{
		{
			name:         "valid mp4 and jpeg",
			method:       http.MethodPost,
			includeVideo: true,
			includeThumb: true,
			returns: []mockSaveReturn{
				{filename: "video.mp4"},
				{filename: "thumbnail.jpg"},
			},
			wantStatus:      http.StatusCreated,
			wantBody:        "video uploaded",
			wantAllowedMIME: []string{"video/mp4", "image/jpeg"},
		},
		{
			name:         "missing video field",
			method:       http.MethodPost,
			includeThumb: true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing video file\n",
		},
		{
			name:         "missing thumbnail field",
			method:       http.MethodPost,
			includeVideo: true,
			wantStatus:   http.StatusBadRequest,
			wantBody:     "missing thumbnail file\n",
		},
		{
			name:         "wrong video MIME",
			method:       http.MethodPost,
			includeVideo: true,
			includeThumb: true,
			returns: []mockSaveReturn{
				{err: &service.ErrInvalidMIMEType{Got: "image/jpeg"}},
			},
			wantStatus: http.StatusBadRequest,
			wantBody: (&service.ErrInvalidMIMEType{
				Got: "image/jpeg",
			}).Error() + "\n",
			wantAllowedMIME: []string{"video/mp4"},
		},
		{
			name:          "wrong thumbnail MIME — cleanup verified",
			method:        http.MethodPost,
			includeVideo:  true,
			includeThumb:  true,
			useUploadsDir: true,
			returns: []mockSaveReturn{
				{filename: "saved-video.mp4"},
				{err: &service.ErrInvalidMIMEType{}},
			},
			wantStatus:      http.StatusBadRequest,
			wantBody:        (&service.ErrInvalidMIMEType{}).Error() + "\n",
			wantAllowedMIME: []string{"video/mp4", "image/jpeg"},
			wantCleanup:     true,
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
			h := handler.NewVideo(saver, uploadsDir)
			r := newVideoRequest(t, tt.method, requestOptions{
				includeVideo: tt.includeVideo,
				includeThumb: tt.includeThumb,
				invalidBody:  tt.invalidBody,
			})
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}

			assertSaveCalls(t, saver.calls, tt.wantAllowedMIME)

			if tt.wantCleanup {
				path := filepath.Join(uploadsDir, tt.returns[0].filename)
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("cleanup stat error = %v, want not exists", err)
				}
			}
		})
	}
}

type requestOptions struct {
	includeVideo bool
	includeThumb bool
	invalidBody  bool
}

func newVideoRequest(
	t *testing.T,
	method string,
	options requestOptions,
) *http.Request {
	t.Helper()

	if options.invalidBody {
		r := httptest.NewRequest(method, "/videos", bytes.NewBufferString("bad"))
		r.Header.Set("Content-Type", "text/plain")

		return r
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if options.includeVideo {
		writeFormFile(t, writer, "video", "video.mp4", "video data")
	}

	if options.includeThumb {
		writeFormFile(t, writer, "thumbnail", "thumbnail.jpg", "thumb data")
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	r := httptest.NewRequest(method, "/videos", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())

	return r
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

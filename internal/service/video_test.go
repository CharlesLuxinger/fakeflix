package service_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

func TestVideoService_SaveFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		contentType string
		allowedMIME string
		content     []byte
		pattern     string
		wantInvalid bool
	}{
		{
			name:        "valid mp4 header",
			filename:    "movie.mp4",
			contentType: service.MIMEVideoMP4,
			allowedMIME: service.MIMEVideoMP4,
			content:     []byte("fake mp4 bytes"),
			pattern:     `^\d{13}-[0-9a-f-]{36}\.mp4$`,
		},
		{
			name:        "valid jpeg header",
			filename:    "poster.jpg",
			contentType: service.MIMEImageJPEG,
			allowedMIME: service.MIMEImageJPEG,
			content:     []byte("fake jpeg bytes"),
			pattern:     `^\d{13}-[0-9a-f-]{36}\.jpg$`,
		},
		{
			name:        "wrong MIME",
			filename:    "movie.mp4",
			contentType: "text/plain",
			allowedMIME: service.MIMEVideoMP4,
			content:     []byte("fake mp4 bytes"),
			wantInvalid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fh := newFileHeader(t, tc.filename, tc.contentType, tc.content)
			uploadsDir := t.TempDir()
			videoService := service.NewVideoService(uploadsDir)

			got, err := videoService.SaveFile(fh, tc.allowedMIME)

			if tc.wantInvalid {
				var invalidMIME *service.ErrInvalidMIMEType
				if !errors.As(err, &invalidMIME) {
					t.Fatalf("expected ErrInvalidMIMEType, got %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("SaveFile() error = %v", err)
			}

			matched := regexp.MustCompile(tc.pattern).MatchString(got)
			if !matched {
				t.Fatalf("filename %q does not match %q", got, tc.pattern)
			}

			saved, err := os.ReadFile(filepath.Join(uploadsDir, got)) //nolint:gosec // test reads back its own written file
			if err != nil {
				t.Fatalf("read saved file: %v", err)
			}

			if !bytes.Equal(saved, tc.content) {
				t.Fatalf("saved bytes = %q, want %q", saved, tc.content)
			}
		})
	}
}

func newFileHeader(t *testing.T, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	partHeader.Set("Content-Type", contentType)

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())

	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read multipart form: %v", err)
	}

	t.Cleanup(func() {
		_ = form.RemoveAll()
	})

	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("form files length = %d, want 1", len(files))
	}

	return files[0]
}

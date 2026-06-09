// Package service provides business logic for video upload operations.
package service

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	// MIMEVideoMP4 is the MIME type for MP4 video files.
	MIMEVideoMP4 = "video/mp4"
	// MIMEImageJPEG is the MIME type for JPEG image files.
	MIMEImageJPEG = "image/jpeg"
	uploadsDir    = "./uploads"
)

// ErrInvalidMIMEType is returned when a file's Content-Type does not match the expected MIME.
//
//nolint:errname // intentional name matching the domain language
type ErrInvalidMIMEType struct {
	Got string
}

func (e *ErrInvalidMIMEType) Error() string {
	return fmt.Sprintf("invalid file type %q: only video/mp4 and image/jpeg are accepted", e.Got)
}

// VideoService handles file storage operations for video uploads.
type VideoService struct {
	uploadsDir string
}

// NewVideoService creates a VideoService with the specified upload directory.
func NewVideoService(dir string) *VideoService {
	return &VideoService{
		uploadsDir: dir,
	}
}

// DefaultVideoService creates a VideoService using the default "./uploads" directory.
func DefaultVideoService() *VideoService {
	return NewVideoService(uploadsDir)
}

// SaveFile validates the MIME type of a multipart file and saves it to disk.
// It returns the generated filename on success.
func (s *VideoService) SaveFile(fh *multipart.FileHeader, allowedMIME string) (string, error) {
	contentType := fh.Header.Get("Content-Type")
	if contentType != allowedMIME {
		return "", &ErrInvalidMIMEType{Got: contentType}
	}

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open multipart file: %w", err)
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	ext := filepath.Ext(fh.Filename)
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixMilli(), uuid.New().String(), ext)
	destPath := filepath.Join(s.uploadsDir, filename)

	if err := os.MkdirAll(s.uploadsDir, 0o750); err != nil {
		return "", fmt.Errorf("create uploads directory: %w", err)
	}

	dst, err := os.Create(destPath) //nolint:gosec // path constructed from controlled inputs
	if err != nil {
		return "", fmt.Errorf("create destination file: %w", err)
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	if _, err := dst.ReadFrom(src); err != nil {
		return "", fmt.Errorf("write destination file: %w", err)
	}

	return filename, nil
}

// Package handler provides HTTP handlers for the FakeFlix API.
package handler

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

type videoSaver interface {
	SaveFile(fh *multipart.FileHeader, allowedMIME string) (string, error)
}

// Video handles POST /video requests for file uploads.
type Video struct {
	saver      videoSaver
	uploadsDir string
}

// NewVideo creates a Video handler with the given file saver and upload directory.
func NewVideo(saver videoSaver, uploadsDir string) *Video {
	return &Video{
		saver:      saver,
		uploadsDir: uploadsDir,
	}
}

func (v *Video) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20) // 32 MB limit

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid multipart body", http.StatusBadRequest)

		return
	}

	videoFiles := r.MultipartForm.File["video"]
	if len(videoFiles) == 0 {
		http.Error(w, "missing video file", http.StatusBadRequest)

		return
	}

	thumbFiles := r.MultipartForm.File["thumbnail"]
	if len(thumbFiles) == 0 {
		http.Error(w, "missing thumbnail file", http.StatusBadRequest)

		return
	}

	videoName, err := v.saver.SaveFile(videoFiles[0], "video/mp4")
	if err != nil {
		handleSaveError(w, err, "failed to save video")

		return
	}

	_, err = v.saver.SaveFile(thumbFiles[0], "image/jpeg")
	if err != nil {
		if removeErr := os.Remove(filepath.Join(v.uploadsDir, filepath.Base(videoName))); removeErr != nil {
			log.Printf("failed to remove video after thumbnail save failure: %v", removeErr)
		}

		handleSaveError(w, err, "failed to save thumbnail")

		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprint(w, "video uploaded")
}

func handleSaveError(w http.ResponseWriter, err error, fallback string) {
	var mimeErr *service.ErrInvalidMIMEType
	if errors.As(err, &mimeErr) {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	http.Error(w, fallback, http.StatusInternalServerError)
}

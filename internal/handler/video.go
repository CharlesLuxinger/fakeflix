// Package handler provides HTTP handlers for the FakeFlix API.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
	"github.com/google/uuid"
)

type videoSaver interface {
	SaveFile(fh *multipart.FileHeader, allowedMIME string) (string, error)
}

type videoCreator interface {
	Create(ctx context.Context, video *model.Video) error
}

// Video handles POST /video requests for file uploads.
type Video struct {
	saver      videoSaver
	repo       videoCreator
	uploadsDir string
}

// NewVideo creates a Video handler with the given file saver, repository, and upload directory.
func NewVideo(saver videoSaver, repo videoCreator, uploadsDir string) *Video {
	return &Video{
		saver:      saver,
		repo:       repo,
		uploadsDir: uploadsDir,
	}
}

//nolint:gocyclo // complexity comes from validation steps inherent to multipart upload handler
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

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "missing or blank title", http.StatusBadRequest)

		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	if description == "" {
		http.Error(w, "missing or blank description", http.StatusBadRequest)

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

	thumbnailName, err := v.saver.SaveFile(thumbFiles[0], "image/jpeg")
	if err != nil {
		if removeErr := os.Remove(filepath.Join(v.uploadsDir, filepath.Base(videoName))); removeErr != nil {
			log.Printf("failed to remove video after thumbnail save failure: %v", removeErr)
		}

		handleSaveError(w, err, "failed to save thumbnail")

		return
	}

	stat, err := os.Stat(filepath.Join(v.uploadsDir, filepath.Base(videoName)))
	if err != nil {
		http.Error(w, "failed to read saved video", http.StatusInternalServerError)

		return
	}

	video := &model.Video{
		ID:           uuid.New().String(),
		Title:        title,
		Description:  description,
		URL:          "uploads/" + filepath.Base(videoName),
		ThumbnailURL: "uploads/" + filepath.Base(thumbnailName),
		SizeInKB:     int(stat.Size() / 1024),
		Duration:     100,
	}

	if err := v.repo.Create(r.Context(), video); err != nil {
		if removeErr := os.Remove(filepath.Join(v.uploadsDir, filepath.Base(videoName))); removeErr != nil {
			log.Printf("failed to remove video after db failure: %v", removeErr)
		}

		if removeErr := os.Remove(filepath.Join(v.uploadsDir, filepath.Base(thumbnailName))); removeErr != nil {
			log.Printf("failed to remove thumbnail after db failure: %v", removeErr)
		}

		http.Error(w, "failed to save video record", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(video); err != nil {
		log.Printf("encode video: %v", err)
	}
}

func handleSaveError(w http.ResponseWriter, err error, fallback string) {
	if _, ok := errors.AsType[*service.ErrInvalidMIMEType](err); ok {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	http.Error(w, fallback, http.StatusInternalServerError)
}

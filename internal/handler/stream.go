package handler

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
)

// videoFinder abstracts video lookup for the stream handler.
type videoFinder interface {
	FindByID(ctx context.Context, id string) (*model.Video, error)
}

// VideoStream handles GET /stream/{videoId} requests with HTTP range support.
type VideoStream struct {
	repo       videoFinder
	uploadsDir string
}

// NewVideoStream creates a VideoStream handler backed by the given repository and uploads directory.
func NewVideoStream(repo videoFinder, uploadsDir string) *VideoStream {
	return &VideoStream{
		repo:       repo,
		uploadsDir: uploadsDir,
	}
}

// ServeHTTP streams a video file with HTTP range support.
func (vs *VideoStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	videoID := r.PathValue("videoId")
	if videoID == "" {
		http.Error(w, "missing video id", http.StatusNotFound)

		return
	}

	video, err := vs.repo.FindByID(r.Context(), videoID)
	if err != nil {
		if errors.Is(err, repository.ErrVideoNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

			return
		}

		//nolint:gosec,nolintlint // G706 suppressed: server-side diagnostic log, values quoted with %q
		log.Printf("find video %q: %v", videoID, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	filePath := filepath.Join(vs.uploadsDir, filepath.Base(video.URL))

	//nolint:gosec // filePath is constrained to uploadsDir by filepath.Base(video.URL)
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)

			return
		}

		//nolint:gosec,nolintlint // G706 suppressed: server-side diagnostic log, values quoted with %q
		log.Printf("open video file %q: %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			//nolint:gosec,nolintlint // G706 suppressed: server-side diagnostic log, values quoted with %q
			log.Printf("close video file %q: %v", filePath, closeErr)
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		//nolint:gosec,nolintlint // G706 suppressed: server-side diagnostic log, values quoted with %q
		log.Printf("stat video file %q: %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, r, filepath.Base(video.URL), stat.ModTime(), file)
}

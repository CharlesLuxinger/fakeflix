package repository_test

import (
	"errors"
	"os"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/database"
	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestNewVideoRepository(t *testing.T) {
	t.Parallel()

	db := &gorm.DB{}
	repo := repository.NewVideoRepository(db)

	if repo == nil {
		t.Fatal("NewVideoRepository returned nil")
	}
}

//nolint:gocyclo,paralleltest // uses shared DB, not safe for parallel runs
func TestVideoRepository_FindByID(t *testing.T) {
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

	videoID := uuid.New().String()
	video := &model.Video{
		ID:           videoID,
		Title:        "Test Video",
		Description:  "Test Description for FindByID",
		URL:          "uploads/test.mp4",
		ThumbnailURL: "uploads/test.jpg",
		SizeInKB:     1024,
		Duration:     120,
	}

	repo := repository.NewVideoRepository(db)

	if err := repo.Create(t.Context(), video); err != nil {
		t.Fatalf("create video: %v", err)
	}

	t.Run("existing video returns full record", func(t *testing.T) {
		got, err := repo.FindByID(t.Context(), videoID)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}

		if got.ID != videoID {
			t.Errorf("ID = %q, want %q", got.ID, videoID)
		}

		if got.Title != "Test Video" {
			t.Errorf("Title = %q, want %q", got.Title, "Test Video")
		}

		if got.Description != "Test Description for FindByID" {
			t.Errorf("Description = %q, want %q", got.Description, "Test Description for FindByID")
		}

		if got.URL != "uploads/test.mp4" {
			t.Errorf("URL = %q, want %q", got.URL, "uploads/test.mp4")
		}

		if got.ThumbnailURL != "uploads/test.jpg" {
			t.Errorf("ThumbnailURL = %q, want %q", got.ThumbnailURL, "uploads/test.jpg")
		}

		if got.SizeInKB != 1024 {
			t.Errorf("SizeInKB = %d, want %d", got.SizeInKB, 1024)
		}

		if got.Duration != 120 {
			t.Errorf("Duration = %d, want %d", got.Duration, 120)
		}
	})

	t.Run("missing video returns ErrVideoNotFound", func(t *testing.T) {
		missingID := uuid.New().String()

		got, err := repo.FindByID(t.Context(), missingID)
		if !errors.Is(err, repository.ErrVideoNotFound) {
			t.Errorf("error = %v, want ErrVideoNotFound", err)
		}

		if got != nil {
			t.Errorf("got video = %v, want nil", got)
		}
	})
}

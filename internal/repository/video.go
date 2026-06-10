// Package repository provides data access interfaces and implementations.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"gorm.io/gorm"
)

// ErrVideoNotFound is returned when a video record is not found in the database.
var ErrVideoNotFound = errors.New("video not found")

// VideoRepository defines data access operations for video records.
type VideoRepository interface {
	Create(ctx context.Context, video *model.Video) error
	FindByID(ctx context.Context, id string) (*model.Video, error)
}

type videoRepository struct {
	db *gorm.DB
}

// NewVideoRepository creates a VideoRepository backed by the given GORM DB.
func NewVideoRepository(db *gorm.DB) VideoRepository {
	return &videoRepository{db: db}
}

func (r *videoRepository) Create(ctx context.Context, video *model.Video) error {
	if err := r.db.WithContext(ctx).Create(video).Error; err != nil {
		return fmt.Errorf("create video: %w", err)
	}

	return nil
}

func (r *videoRepository) FindByID(ctx context.Context, id string) (*model.Video, error) {
	var video model.Video

	if err := r.db.WithContext(ctx).First(&video, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find video by id %q: %w", id, ErrVideoNotFound)
		}

		return nil, fmt.Errorf("find video by id %q: %w", id, err)
	}

	return &video, nil
}

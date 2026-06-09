// Package repository provides data access interfaces and implementations.
package repository

import (
	"context"
	"fmt"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"gorm.io/gorm"
)

// VideoRepository defines data access operations for video records.
type VideoRepository interface {
	Create(ctx context.Context, video *model.Video) error
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

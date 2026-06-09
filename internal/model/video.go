// Package model provides domain types for video metadata.
package model

import "time"

// Video represents a video record persisted in PostgreSQL.
type Video struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnailUrl" gorm:"column:thumbnail_url"`
	SizeInKB     int       `json:"sizeInKb" gorm:"column:size_in_kb"`
	Duration     int       `json:"duration"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

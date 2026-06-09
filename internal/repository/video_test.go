package repository_test

import (
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/repository"
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

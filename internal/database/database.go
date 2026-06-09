// Package database provides PostgreSQL connection bootstrapping.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open connects to PostgreSQL, runs AutoMigrate, and returns a cleanup function.
func Open(ctx context.Context, dsn string) (*gorm.DB, func() error, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	if err := db.AutoMigrate(&model.Video{}); err != nil {
		return nil, nil, fmt.Errorf("auto migrate: %w", err)
	}

	return db, sqlDB.Close, nil
}

// Package main is the entry point for the FakeFlix server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CharlesLuxinger/fakeflix/internal/database"
	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/repository"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

func main() {
	if err := os.MkdirAll("./uploads", 0o750); err != nil {
		log.Fatalf("create uploads directory: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL not set")
	}

	ctx := context.Background()

	db, closeDB, err := database.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	defer func() {
		if err := closeDB(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	videoSvc := service.DefaultVideoService()
	videoRepo := repository.NewVideoRepository(db)
	healthHandler := handler.NewHealth()
	videoHandler := handler.NewVideo(videoSvc, videoRepo, "./uploads")

	mux := http.NewServeMux()
	mux.Handle("GET /", healthHandler)
	mux.Handle("POST /video", videoHandler)

	srv := &http.Server{
		Addr:         ":3000",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on :3000")

		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := srv.Shutdown(ctx); err != nil {
		cancel()
		log.Printf("shutdown: %v", err)

		return
	}

	cancel()

	log.Printf("server stopped")
}

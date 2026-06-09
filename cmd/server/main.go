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

	"github.com/CharlesLuxinger/fakeflix/internal/handler"
	"github.com/CharlesLuxinger/fakeflix/internal/service"
)

func main() {
	if err := os.MkdirAll("./uploads", 0o750); err != nil {
		log.Fatalf("create uploads directory: %v", err)
	}

	videoSvc := service.DefaultVideoService()
	healthHandler := handler.NewHealth()
	videoHandler := handler.NewVideo(videoSvc, "./uploads")

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
		log.Fatalf("shutdown: %v", err)
	}

	cancel()

	log.Printf("server stopped")
}

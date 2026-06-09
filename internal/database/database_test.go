package database_test

import (
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/database"
)

func TestOpen_InvalidDSN(t *testing.T) {
	t.Parallel()

	_, _, err := database.Open(t.Context(), "invalid-dsn")
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

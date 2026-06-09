// Package handler provides HTTP handlers for the FakeFlix API.
package handler

import (
	"fmt"
	"net/http"
)

// Ensure Health implements http.Handler.
var _ http.Handler = (*Health)(nil)

// Health handles GET / endpoint.
type Health struct{}

// NewHealth creates a new Health handler.
func NewHealth() *Health {
	return &Health{}
}

// ServeHTTP implements http.Handler.
func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "Hello World!")
}

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CharlesLuxinger/fakeflix/internal/handler"
)

func TestHealth_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET returns 200",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "Hello World!",
		},
		{
			name:       "POST returns 405",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   http.StatusText(http.StatusMethodNotAllowed) + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := handler.NewHealth()
			r := httptest.NewRequestWithContext(t.Context(), tt.method, "/", nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

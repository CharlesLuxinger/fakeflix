package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CharlesLuxinger/fakeflix/internal/model"
)

func TestVideoJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)

	video := model.Video{
		ID:           "123",
		Title:        "Test Title",
		Description:  "Test Description",
		URL:          "http://example.com/video.mp4",
		ThumbnailURL: "http://example.com/thumb.jpg",
		SizeInKB:     1024,
		Duration:     120,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(video)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"id", "123"},
		{"title", "Test Title"},
		{"description", "Test Description"},
		{"url", "http://example.com/video.mp4"},
		{"thumbnailUrl", "http://example.com/thumb.jpg"},
		{"sizeInKb", float64(1024)},
		{"duration", float64(120)},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := result[tt.key]
			if !ok {
				t.Errorf("missing key %q in JSON output", tt.key)
				return
			}

			if got != tt.expected {
				t.Errorf("key %q: got %v (%T), want %v (%T)", tt.key, got, got, tt.expected, tt.expected)
			}
		})
	}

	for _, key := range []string{"createdAt", "updatedAt"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			got, ok := result[key]
			if !ok {
				t.Errorf("missing key %q in JSON output", key)

				return
			}

			str, ok := got.(string)
			if !ok {
				t.Errorf("key %q is not a string: got %T", key, got)

				return
			}

			if _, err := time.Parse(time.RFC3339Nano, str); err != nil {
				t.Errorf("key %q is not a valid time: %v", key, err)
			}
		})
	}
}

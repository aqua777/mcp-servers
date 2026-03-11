package fetch

import (
	"strings"
	"testing"
)

func TestApplyChunking(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		startIndex int
		maxLength  int
		want       string
	}{
		{
			name:       "no truncation needed",
			content:    "Hello, World!",
			startIndex: 0,
			maxLength:  100,
			want:       "Hello, World!",
		},
		{
			name:       "truncation with continuation hint",
			content:    "This is a very long content that needs to be truncated",
			startIndex: 0,
			maxLength:  20,
			want:       "This is a very long \n\n<error>Content truncated. Call the fetch tool with a start_index of 20 to get more content.</error>",
		},
		{
			name:       "start index beyond content",
			content:    "Short content",
			startIndex: 100,
			maxLength:  50,
			want:       "<error>No more content available.</error>",
		},
		{
			name:       "start index at end of content",
			content:    "Exact length",
			startIndex: 12,
			maxLength:  50,
			want:       "<error>No more content available.</error>",
		},
		{
			name:       "continuation from middle",
			content:    "First part. Second part. Third part.",
			startIndex: 12,
			maxLength:  15,
			want:       "Second part. Th\n\n<error>Content truncated. Call the fetch tool with a start_index of 27 to get more content.</error>",
		},
		{
			name:       "exact match no continuation",
			content:    "Exactly twenty chars",
			startIndex: 0,
			maxLength:  20,
			want:       "Exactly twenty chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyChunking(tt.content, tt.startIndex, tt.maxLength)
			if got != tt.want {
				t.Errorf("applyChunking() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyChunkingBoundaries(t *testing.T) {
	content := strings.Repeat("a", 1000)

	result := applyChunking(content, 0, 500)
	if len(result) <= 500 {
		t.Errorf("Expected continuation hint to be added, got length %d", len(result))
	}
	if !strings.Contains(result, "start_index of 500") {
		t.Errorf("Expected continuation hint with start_index of 500, got: %s", result)
	}

	result = applyChunking(content, 500, 500)
	if len(result) != 500 {
		t.Errorf("Expected exactly 500 chars (no continuation hint at end), got length %d", len(result))
	}
	if strings.Contains(result, "<error>") {
		t.Errorf("Expected no error message when fetching last chunk, got: %s", result)
	}
}

func TestApplyChunking_Empty(t *testing.T) {
	got := applyChunking("abc", 0, 0)
	if got != "<error>No more content available.</error>" {
		t.Errorf("Expected error for empty content, got: %s", got)
	}
}

func TestApplyChunking_MaxLengthGreaterThanLength(t *testing.T) {
	got := applyChunking("abc", 0, 10)
	if got != "abc" {
		t.Errorf("Expected abc, got: %s", got)
	}
}

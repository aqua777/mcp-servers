package fetch

import (
	"testing"
)

func TestGetRobotsTxtURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple URL",
			url:      "https://example.com/page",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
		{
			name:     "URL with path",
			url:      "https://example.com/some/deep/path/page.html",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
		{
			name:     "URL with query params",
			url:      "https://example.com/page?foo=bar&baz=qux",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
		{
			name:     "URL with port",
			url:      "https://example.com:8080/page",
			expected: "https://example.com:8080/robots.txt",
			wantErr:  false,
		},
		{
			name:     "URL with fragment",
			url:      "https://example.com/page#section",
			expected: "https://example.com/robots.txt",
			wantErr:  false,
		},
		{
			name:     "HTTP URL",
			url:      "http://example.com/page",
			expected: "http://example.com/robots.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getRobotsTxtURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRobotsTxtURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("getRobotsTxtURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProcessRobotsTxt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "removes comments",
			input: `User-agent: *
# This is a comment
Disallow: /admin
# Another comment
Allow: /public`,
			expected: `User-agent: *
Disallow: /admin
Allow: /public`,
		},
		{
			name: "keeps non-comment lines",
			input: `User-agent: *
Disallow: /
Allow: /public`,
			expected: `User-agent: *
Disallow: /
Allow: /public`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processRobotsTxt(tt.input)
			if result != tt.expected {
				t.Errorf("processRobotsTxt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

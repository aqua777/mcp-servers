package filesystem

import (
	"path/filepath"
	"testing"
)

func TestIsPathWithinAllowedDirectories(t *testing.T) {
	// Base directories for testing
	tmpDir := filepath.Clean("/tmp")
	usrDir := filepath.Clean("/usr")

	tests := []struct {
		name       string
		path       string
		allowed    []string
		want       bool
		wantErr    bool
	}{
		{
			name:    "Empty path",
			path:    "",
			allowed: []string{tmpDir},
			want:    false,
			wantErr: false,
		},
		{
			name:    "Empty allowed",
			path:    filepath.Join(tmpDir, "file.txt"),
			allowed: []string{},
			want:    false,
			wantErr: false,
		},
		{
			name:    "Null byte in path",
			path:    filepath.Join(tmpDir, "file\x00.txt"),
			allowed: []string{tmpDir},
			want:    false,
			wantErr: false,
		},
		{
			name:    "Relative path",
			path:    "relative/path.txt",
			allowed: []string{tmpDir},
			want:    false,
			wantErr: true,
		},
		{
			name:    "Exact match",
			path:    tmpDir,
			allowed: []string{tmpDir},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Subdirectory match",
			path:    filepath.Join(tmpDir, "subdir", "file.txt"),
			allowed: []string{tmpDir},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Outside directory",
			path:    filepath.Join(usrDir, "file.txt"),
			allowed: []string{tmpDir},
			want:    false,
			wantErr: false,
		},
		{
			name:    "Partial prefix match (should fail)",
			path:    tmpDir + "2", // e.g. /tmp2
			allowed: []string{tmpDir}, // /tmp
			want:    false,
			wantErr: false,
		},
		{
			name:    "Directory traversal attempt (should fail)",
			path:    filepath.Join(tmpDir, "..", "usr", "file.txt"),
			allowed: []string{tmpDir},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsPathWithinAllowedDirectories(tt.path, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsPathWithinAllowedDirectories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsPathWithinAllowedDirectories() = %v, want %v", got, tt.want)
			}
		})
	}
}

package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateRepoPath checks that repoPath is within allowedRepository.
// When allowedRepository is empty, any path is permitted.
// Resolves symlinks on both paths to prevent traversal attacks.
func validateRepoPath(repoPath, allowedRepository string) error {
	if allowedRepository == "" {
		return nil
	}

	resolvedRepo, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		// Path may not exist yet or symlink resolution failed
		return fmt.Errorf("invalid repository path %q: %w", repoPath, err)
	}

	resolvedAllowed, err := filepath.EvalSymlinks(allowedRepository)
	if err != nil {
		return fmt.Errorf("invalid allowed repository path %q: %w", allowedRepository, err)
	}

	resolvedRepo = filepath.Clean(resolvedRepo)
	resolvedAllowed = filepath.Clean(resolvedAllowed)

	// Accept exact match or sub-directory (must have trailing separator to avoid prefix collisions)
	if resolvedRepo != resolvedAllowed && !strings.HasPrefix(resolvedRepo, resolvedAllowed+string(filepath.Separator)) {
		return fmt.Errorf("repository path %q is outside the allowed repository %q", repoPath, allowedRepository)
	}

	return nil
}

// validateRefName rejects ref names that start with '-' to prevent flag injection
// into git operations (defense in depth matching the Python reference behavior).
func validateRefName(name string) error {
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid ref name %q: cannot start with '-'", name)
	}
	return nil
}

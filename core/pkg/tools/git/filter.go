package git

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// defaultIncludePatterns is the include list used when the caller does not
// supply any include_patterns. It matches every file inside the repository.
var defaultIncludePatterns = []string{"**/*"}

// FileFilter encapsulates the include/exclude model for file paths returned by
// git operations.
//
// Model:
//   - includePatterns: files must match at least one pattern (default: ["**/*"])
//   - excludePatterns (gitignore + user-supplied): files must not match any pattern
//   - Final decision: included AND NOT excluded
//
// The gitignore matcher is the gitignore-aware evaluator for the exclusion side;
// it represents the .gitignore file(s) content as the default exclusion list.
type FileFilter struct {
	gitignoreMatcher gitignore.Matcher
	includePatterns  []string
	excludePatterns  []string
}

// NewFileFilter builds a FileFilter from a worktree.
//
// include defaults to ["**/*"] when nil or empty — meaning all files pass.
// exclude defaults to [] when nil or empty; when noGitignore is false, the
// .gitignore file(s) in the worktree are loaded and treated as the default
// exclusion list (merged with any caller-supplied exclude patterns).
func NewFileFilter(wt *gogit.Worktree, noGitignore bool, include, exclude []string) (*FileFilter, error) {
	// Default include: match everything
	if len(include) == 0 {
		include = defaultIncludePatterns
	}

	f := &FileFilter{
		includePatterns: include,
		excludePatterns: exclude,
	}

	// Default exclusion: load .gitignore pattern(s) unless opted out
	if !noGitignore {
		patterns, err := gitignore.ReadPatterns(wt.Filesystem, nil)
		if err != nil {
			return nil, err
		}
		if len(patterns) > 0 {
			f.gitignoreMatcher = gitignore.NewMatcher(patterns)
		}
	}

	return f, nil
}

// IsEmpty returns true when the filter is equivalent to the defaults and will
// not exclude any file — include is the default ["**/*"], no gitignore matcher
// is loaded, and no additional exclude patterns are set.
// Callers can use this to skip filtering overhead entirely.
func (f *FileFilter) IsEmpty() bool {
	if f.gitignoreMatcher != nil || len(f.excludePatterns) > 0 {
		return false
	}
	// Include is non-empty by construction (defaultIncludePatterns is set in
	// NewFileFilter). Consider it empty (no-op) only when the sole pattern is
	// the default wildcard — i.e. nothing was actually filtered.
	if len(f.includePatterns) == 1 && f.includePatterns[0] == defaultIncludePatterns[0] {
		return true
	}
	return len(f.includePatterns) == 0
}

// Match returns true if the given repo-relative path should be INCLUDED in results.
// path uses forward slashes. isDir indicates whether the path refers to a directory.
//
// Decision: INCLUDED = matches include AND does NOT match exclude.
//
//  1. Include check: path must match at least one includePattern.
//     Default includePatterns is ["**/*"] so all files pass when the caller
//     did not supply explicit patterns.
//  2. Exclude check (gitignore): if the gitignore matcher is loaded and the
//     path matches, it is excluded.
//  3. Exclude check (user patterns): path must not match any excludePattern.
func (f *FileFilter) Match(path string, isDir bool) bool {
	// 1. Include — must match at least one pattern
	if len(f.includePatterns) > 0 {
		matched := false
		for _, pat := range f.includePatterns {
			if ok, err := doublestar.Match(pat, path); err == nil && ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 2. Exclude — gitignore (default exclusion list from .gitignore files)
	if f.gitignoreMatcher != nil {
		parts := strings.Split(path, "/")
		if f.gitignoreMatcher.Match(parts, isDir) {
			return false
		}
	}

	// 3. Exclude — user-supplied glob patterns
	for _, pat := range f.excludePatterns {
		if ok, err := doublestar.Match(pat, path); err == nil && ok {
			return false
		}
	}

	return true
}

// filterDiffFiles removes DiffFile entries from result that do not match the filter.
// It recalculates the Summary and clears RawText so text formatters regenerate
// output from the filtered Files slice instead of the pre-filter raw diff.
// When filter is nil or empty, this is a no-op.
func filterDiffFiles(result *DiffResult, filter *FileFilter) {
	if filter == nil || filter.IsEmpty() {
		return
	}

	filtered := result.Files[:0]
	for _, f := range result.Files {
		if filter.Match(f.Path, false) {
			filtered = append(filtered, f)
		}
	}
	result.Files = filtered

	// Recalculate summary
	result.Summary = DiffSummary{TotalFiles: len(result.Files)}
	for _, f := range result.Files {
		result.Summary.TotalAdditions += f.Additions
		result.Summary.TotalDeletions += f.Deletions
	}

	// Clear RawText so the text formatter regenerates from the filtered Files slice
	result.RawText = ""

	if len(result.Files) == 0 {
		result.Status = "no_changes"
	}
}

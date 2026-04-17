package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/suite"
)

type filterTestSuite struct {
	suite.Suite
	repoDir string
	wt      *gogit.Worktree
}

func TestFilterSuite(t *testing.T) {
	suite.Run(t, new(filterTestSuite))
}

func (s *filterTestSuite) SetupTest() {
	dir, err := os.MkdirTemp("", "git-filter-test-*")
	s.Require().NoError(err)
	s.repoDir = dir

	repo, err := gogit.PlainInit(dir, false)
	s.Require().NoError(err)

	wt, err := repo.Worktree()
	s.Require().NoError(err)
	s.wt = wt

	// Create a .gitignore with some patterns
	s.writeFile(".gitignore", "*.log\nbuild/\n")

	// Create initial commit so the repo is valid
	_, err = wt.Add(".gitignore")
	s.Require().NoError(err)
	_, err = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t.com"},
	})
	s.Require().NoError(err)
}

func (s *filterTestSuite) TearDownTest() {
	os.RemoveAll(s.repoDir)
}

func (s *filterTestSuite) writeFile(name, content string) {
	path := filepath.Join(s.repoDir, filepath.FromSlash(name))
	dir := filepath.Dir(path)
	s.Require().NoError(os.MkdirAll(dir, 0o755))
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
}

// TestIsEmpty — filter with no options is empty.
func (s *filterTestSuite) TestIsEmpty() {
	f, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)
	s.True(f.IsEmpty())
}

// TestIsEmptyWithGitignore — filter with gitignore active is NOT empty.
func (s *filterTestSuite) TestIsEmptyWithGitignore() {
	f, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)
	s.False(f.IsEmpty())
}

// TestIsEmptyWithInclude — filter with include patterns is NOT empty.
func (s *filterTestSuite) TestIsEmptyWithInclude() {
	f, err := NewFileFilter(s.wt, true, []string{"*.go"}, nil)
	s.Require().NoError(err)
	s.False(f.IsEmpty())
}

// TestIsEmptyWithExclude — filter with exclude patterns is NOT empty.
func (s *filterTestSuite) TestIsEmptyWithExclude() {
	f, err := NewFileFilter(s.wt, true, nil, []string{"vendor/**"})
	s.Require().NoError(err)
	s.False(f.IsEmpty())
}

// TestGitignoreExcludesLogFile — *.log in .gitignore should exclude app.log.
func (s *filterTestSuite) TestGitignoreExcludesLogFile() {
	f, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)

	s.False(f.Match("app.log", false))
	s.False(f.Match("logs/server.log", false))
}

// TestGitignoreExcludesDirectory — build/ in .gitignore should exclude the dir.
func (s *filterTestSuite) TestGitignoreExcludesDirectory() {
	f, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)

	s.False(f.Match("build", true))
}

// TestGitignoreAllowsNormalFiles — non-ignored files pass through.
func (s *filterTestSuite) TestGitignoreAllowsNormalFiles() {
	f, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.True(f.Match("README.md", false))
	s.True(f.Match("internal/server.go", false))
}

// TestNoGitignoreBypassesGitignore — ignored files pass when noGitignore=true.
func (s *filterTestSuite) TestNoGitignoreBypassesGitignore() {
	f, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)

	s.True(f.Match("app.log", false))
	s.True(f.Match("build", true))
}

// TestIncludePatternsWhitelist — only files matching include pass.
func (s *filterTestSuite) TestIncludePatternsWhitelist() {
	f, err := NewFileFilter(s.wt, true, []string{"*.go"}, nil)
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.False(f.Match("README.md", false))
	s.False(f.Match("go.mod", false))
}

// TestIncludeDoubleStar — doublestar pattern matches nested paths.
func (s *filterTestSuite) TestIncludeDoubleStar() {
	f, err := NewFileFilter(s.wt, true, []string{"**/*.go"}, nil)
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.True(f.Match("pkg/server/handler.go", false))
	s.False(f.Match("README.md", false))
}

// TestExcludePatternsBlacklist — files matching exclude are removed.
func (s *filterTestSuite) TestExcludePatternsBlacklist() {
	f, err := NewFileFilter(s.wt, true, nil, []string{"vendor/**"})
	s.Require().NoError(err)

	s.False(f.Match("vendor/github.com/foo/bar.go", false))
	s.True(f.Match("main.go", false))
	s.True(f.Match("internal/pkg/util.go", false))
}

// TestCombinedIncludeExclude — include *.go, exclude *_test.go.
func (s *filterTestSuite) TestCombinedIncludeExclude() {
	f, err := NewFileFilter(s.wt, true, []string{"**/*.go"}, []string{"**/*_test.go"})
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.True(f.Match("pkg/server.go", false))
	s.False(f.Match("main_test.go", false))
	s.False(f.Match("pkg/server_test.go", false))
	s.False(f.Match("README.md", false))
}

// TestCombinedGitignoreAndExclude — gitignore + extra exclude pattern.
func (s *filterTestSuite) TestCombinedGitignoreAndExclude() {
	f, err := NewFileFilter(s.wt, false, nil, []string{"*.tmp"})
	s.Require().NoError(err)

	s.False(f.Match("app.log", false))  // excluded by gitignore
	s.False(f.Match("work.tmp", false)) // excluded by exclude_patterns
	s.True(f.Match("main.go", false))   // passes both
}

// TestMultipleIncludePatterns — OR semantics: any match passes.
func (s *filterTestSuite) TestMultipleIncludePatterns() {
	f, err := NewFileFilter(s.wt, true, []string{"*.go", "*.md"}, nil)
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.True(f.Match("README.md", false))
	s.False(f.Match("config.yaml", false))
}

// TestFilterDiffFiles — filterDiffFiles removes non-matching files and recalculates summary.
func (s *filterTestSuite) TestFilterDiffFiles() {
	result := &DiffResult{
		Status: "has_changes",
		Files: []DiffFile{
			{Path: "main.go", Status: "modified", Additions: 5, Deletions: 2, Changes: []DiffChange{}},
			{Path: "README.md", Status: "modified", Additions: 1, Deletions: 0, Changes: []DiffChange{}},
			{Path: "app.log", Status: "modified", Additions: 3, Deletions: 3, Changes: []DiffChange{}},
		},
		Summary: DiffSummary{TotalFiles: 3, TotalAdditions: 9, TotalDeletions: 5},
		RawText: "raw diff content",
	}

	f, err := NewFileFilter(s.wt, true, []string{"*.go"}, nil)
	s.Require().NoError(err)

	filterDiffFiles(result, f)

	s.Len(result.Files, 1)
	s.Equal("main.go", result.Files[0].Path)
	s.Equal(1, result.Summary.TotalFiles)
	s.Equal(5, result.Summary.TotalAdditions)
	s.Equal(2, result.Summary.TotalDeletions)
	s.Equal("", result.RawText, "RawText must be cleared after filtering")
	s.Equal("has_changes", result.Status)
}

// TestFilterDiffFilesAllExcluded — status becomes no_changes when all files filtered.
func (s *filterTestSuite) TestFilterDiffFilesAllExcluded() {
	result := &DiffResult{
		Status: "has_changes",
		Files: []DiffFile{
			{Path: "app.log", Status: "modified", Additions: 3, Deletions: 3, Changes: []DiffChange{}},
		},
		Summary: DiffSummary{TotalFiles: 1, TotalAdditions: 3, TotalDeletions: 3},
	}

	f, err := NewFileFilter(s.wt, true, []string{"*.go"}, nil)
	s.Require().NoError(err)

	filterDiffFiles(result, f)

	s.Len(result.Files, 0)
	s.Equal("no_changes", result.Status)
	s.Equal(0, result.Summary.TotalFiles)
}

// TestFilterDiffFilesNoop — nil and empty filters are no-ops.
func (s *filterTestSuite) TestFilterDiffFilesNoop() {
	result := &DiffResult{
		Status:  "has_changes",
		Files:   []DiffFile{{Path: "main.go", Status: "modified", Changes: []DiffChange{}}},
		RawText: "raw",
	}

	filterDiffFiles(result, nil)
	s.Equal("raw", result.RawText, "nil filter should not clear RawText")

	emptyFilter, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)
	filterDiffFiles(result, emptyFilter)
	s.Equal("raw", result.RawText, "empty filter should not clear RawText")
}

// ── Default model tests ───────────────────────────────────────────────────────

// TestDefaultIncludeIsWildcard — when no include_patterns supplied, all files pass the include check.
func (s *filterTestSuite) TestDefaultIncludeIsWildcard() {
	// noGitignore=true so only the default include ["**/*"] is in play
	f, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)

	// Default include must pass any file at any nesting depth
	s.True(f.Match("main.go", false))
	s.True(f.Match("cmd/server/main.go", false))
	s.True(f.Match("a/b/c/d/deep.go", false))
	s.True(f.Match("README.md", false))
	s.True(f.Match("Makefile", false))
}

// TestDefaultExcludeEmptyWithoutGitignore — no .gitignore means nothing is excluded by default.
func (s *filterTestSuite) TestDefaultExcludeEmptyWithoutGitignore() {
	// Create a repo WITHOUT a .gitignore
	dir, err := os.MkdirTemp("", "git-filter-nogi-*")
	s.Require().NoError(err)
	defer os.RemoveAll(dir)

	repo, err := gogit.PlainInit(dir, false)
	s.Require().NoError(err)
	wt, err := repo.Worktree()
	s.Require().NoError(err)

	f, err := NewFileFilter(wt, false, nil, nil)
	s.Require().NoError(err)

	// With no .gitignore, gitignoreMatcher is nil → IsEmpty() should be true
	s.True(f.IsEmpty(), "no .gitignore + no patterns = empty (no-op) filter")

	// All files pass
	s.True(f.Match("app.log", false))
	s.True(f.Match("build/output", false))
	s.True(f.Match("vendor/pkg/file.go", false))
}

// TestDefaultExcludeFromGitignore — .gitignore present populates the default exclusion list.
func (s *filterTestSuite) TestDefaultExcludeFromGitignore() {
	// s.wt has .gitignore with "*.log\nbuild/\n"
	f, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)

	// gitignore-populated exclusions
	s.False(f.Match("app.log", false), "*.log matched by gitignore exclusion")
	s.False(f.Match("build", true), "build/ matched by gitignore exclusion")

	// non-ignored files still pass the default include
	s.True(f.Match("main.go", false))
	s.True(f.Match("cmd/server.go", false))
}

// TestNoGitignoreClearsDefaultExclusion — no_gitignore=true means gitignore is NOT loaded as default exclusion.
func (s *filterTestSuite) TestNoGitignoreClearsDefaultExclusion() {
	// s.wt has .gitignore with "*.log\nbuild/\n"
	f, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)

	// gitignore not loaded, so these pass
	s.True(f.Match("app.log", false), "*.log should pass when no_gitignore=true")
	s.True(f.Match("build", true), "build/ should pass when no_gitignore=true")
}

// TestIncludeOverridesDefault — user include_patterns replace the default ["**/*"].
func (s *filterTestSuite) TestIncludeOverridesDefault() {
	f, err := NewFileFilter(s.wt, true, []string{"**/*.go"}, nil)
	s.Require().NoError(err)

	s.True(f.Match("main.go", false))
	s.True(f.Match("pkg/server/handler.go", false))
	// Files not matching **/*.go are excluded by the include filter
	s.False(f.Match("README.md", false))
	s.False(f.Match("schema.sql", false))
}

// TestExcludeUnionWithGitignore — user exclude_patterns are unioned with gitignore exclusions.
func (s *filterTestSuite) TestExcludeUnionWithGitignore() {
	// gitignore: *.log, build/; extra exclude: *.tmp
	f, err := NewFileFilter(s.wt, false, nil, []string{"*.tmp"})
	s.Require().NoError(err)

	s.False(f.Match("app.log", false), "excluded by gitignore")
	s.False(f.Match("work.tmp", false), "excluded by user exclude_patterns")
	s.True(f.Match("main.go", false), "passes include and neither exclusion")
}

// TestIsEmptySemantics — IsEmpty reflects "default include only, no exclusions".
func (s *filterTestSuite) TestIsEmptySemantics() {
	// no_gitignore=true, no patterns → default include ["**/*"], no exclusions → empty
	fEmpty, err := NewFileFilter(s.wt, true, nil, nil)
	s.Require().NoError(err)
	s.True(fEmpty.IsEmpty())

	// gitignore loaded → not empty (has exclusions)
	fGi, err := NewFileFilter(s.wt, false, nil, nil)
	s.Require().NoError(err)
	s.False(fGi.IsEmpty())

	// custom include → not empty
	fInc, err := NewFileFilter(s.wt, true, []string{"*.go"}, nil)
	s.Require().NoError(err)
	s.False(fInc.IsEmpty())

	// custom exclude → not empty
	fExc, err := NewFileFilter(s.wt, true, nil, []string{"vendor/**"})
	s.Require().NoError(err)
	s.False(fExc.IsEmpty())
}

// ──────────────────────────────────────────────────────────────────────────────

// TestInvalidGlobPattern — invalid glob patterns do not panic, just don't match.
func (s *filterTestSuite) TestInvalidGlobPattern() {
	f, err := NewFileFilter(s.wt, true, []string{"["}, nil)
	s.Require().NoError(err)
	// Invalid pattern should not match (doublestar returns error, we treat as no-match)
	s.False(f.Match("main.go", false))
}

// TestInvalidExcludeGlobPattern — invalid exclude glob patterns do not panic.
func (s *filterTestSuite) TestInvalidExcludeGlobPattern() {
	f, err := NewFileFilter(s.wt, true, nil, []string{"["})
	s.Require().NoError(err)
	// Invalid exclude pattern: error means no match, so file is NOT excluded
	s.True(f.Match("main.go", false))
}

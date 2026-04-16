package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

// gitServerTestSuite sets up a real git repo in a temp dir for testing.
type gitServerTestSuite struct {
	suite.Suite
	repoDir       string
	repo          *gogit.Repository
	gs            *GitServer
	defaultBranch string
}

func (s *gitServerTestSuite) SetupTest() {
	dir, err := os.MkdirTemp("", "git-server-test-*")
	s.Require().NoError(err)
	s.repoDir = dir

	// Init repo
	repo, err := gogit.PlainInit(dir, false)
	s.Require().NoError(err)
	s.repo = repo

	// Create initial commit
	s.writeFile("readme.txt", "initial content\n")
	s.stageFile("readme.txt")
	s.commit("initial commit")

	// Record the default branch name
	head, err := repo.Head()
	s.Require().NoError(err)
	s.defaultBranch = head.Name().Short()

	// Create server with no restrictions
	srv, err := NewServer(context.Background(), Options{})
	s.Require().NoError(err)
	s.gs = &GitServer{server: srv, options: Options{}}
}

// helper: make NewServer return *mcp.Server so we can extract it
func init() {
	// nothing — just ensures init() registered the server
}

func (s *gitServerTestSuite) TearDownTest() {
	os.RemoveAll(s.repoDir)
}

func (s *gitServerTestSuite) writeFile(name, content string) {
	path := filepath.Join(s.repoDir, name)
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
}

func (s *gitServerTestSuite) stageFile(name string) {
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	_, err = wt.Add(name)
	s.Require().NoError(err)
}

func (s *gitServerTestSuite) commit(msg string) string {
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	hash, err := wt.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	s.Require().NoError(err)
	return hash.String()
}

// callTool is a test helper that invokes a handler by tool name.
func (s *gitServerTestSuite) callTool(toolName string, args map[string]any) *mcp.CallToolResult {
	argsBytes, err := json.Marshal(args)
	s.Require().NoError(err)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      toolName,
			Arguments: argsBytes,
		},
	}

	ctx := context.Background()
	var result *mcp.CallToolResult
	switch toolName {
	case ToolGitStatus:
		result, err = s.gs.handleGitStatus(ctx, req)
	case ToolGitDiffUnstaged:
		result, err = s.gs.handleGitDiffUnstaged(ctx, req)
	case ToolGitDiffStaged:
		result, err = s.gs.handleGitDiffStaged(ctx, req)
	case ToolGitDiff:
		result, err = s.gs.handleGitDiff(ctx, req)
	case ToolGitCommit:
		result, err = s.gs.handleGitCommit(ctx, req)
	case ToolGitAdd:
		result, err = s.gs.handleGitAdd(ctx, req)
	case ToolGitReset:
		result, err = s.gs.handleGitReset(ctx, req)
	case ToolGitLog:
		result, err = s.gs.handleGitLog(ctx, req)
	case ToolGitCreateBranch:
		result, err = s.gs.handleGitCreateBranch(ctx, req)
	case ToolGitCheckout:
		result, err = s.gs.handleGitCheckout(ctx, req)
	case ToolGitShow:
		result, err = s.gs.handleGitShow(ctx, req)
	case ToolGitBranch:
		result, err = s.gs.handleGitBranch(ctx, req)
	default:
		s.Fail("unknown tool: " + toolName)
	}
	s.Require().NoError(err)
	s.Require().NotNil(result)
	return result
}

func (s *gitServerTestSuite) resultText(r *mcp.CallToolResult) string {
	s.Require().NotEmpty(r.Content)
	return r.Content[0].(*mcp.TextContent).Text
}

// --- NewServer ---

func (s *gitServerTestSuite) TestNewServer_ReturnsServer() {
	srv, err := NewServer(context.Background(), Options{})
	s.NoError(err)
	s.NotNil(srv)
}

func (s *gitServerTestSuite) TestNewServer_InvalidOpts() {
	_, err := NewServer(context.Background(), "not-an-options-struct")
	s.Error(err)
}

func (s *gitServerTestSuite) TestNewServer_NilOpts() {
	srv, err := NewServer(context.Background(), nil)
	s.NoError(err)
	s.NotNil(srv)
}

// --- git_status ---

func (s *gitServerTestSuite) TestGitStatus_CleanRepo() {
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "nothing to commit")
}

func (s *gitServerTestSuite) TestGitStatus_DirtyRepo() {
	s.writeFile("new_file.txt", "some content")
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "new_file.txt")
}

func (s *gitServerTestSuite) TestGitStatus_InvalidRepo() {
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": "/tmp/definitely-not-a-repo-xyz"})
	s.True(result.IsError)
}

// --- git_diff_unstaged ---

func (s *gitServerTestSuite) TestGitDiffUnstaged_Empty() {
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Empty(text)
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_WithChanges() {
	s.writeFile("readme.txt", "modified content\n")
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "readme.txt")
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_WithContextLines() {
	s.writeFile("readme.txt", "modified content\n")
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir, "context_lines": 5})
	s.False(result.IsError)
}

// --- git_diff_staged ---

func (s *gitServerTestSuite) TestGitDiffStaged_Empty() {
	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Empty(text)
}

func (s *gitServerTestSuite) TestGitDiffStaged_WithStagedFile() {
	s.writeFile("staged.txt", "staged content\n")
	s.stageFile("staged.txt")
	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "staged.txt")
}

// --- git_diff ---

func (s *gitServerTestSuite) TestGitDiff_BetweenBranches() {
	// Create a second commit on a new branch
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/feature",
		Create: true,
	})
	s.Require().NoError(err)

	s.writeFile("feature.txt", "feature content\n")
	s.stageFile("feature.txt")
	s.commit("feature commit")

	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": s.repoDir, "target": s.defaultBranch})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "feature.txt")
}

func (s *gitServerTestSuite) TestGitDiff_FlagInjectionRejected() {
	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": s.repoDir, "target": "--help"})
	s.True(result.IsError)
	s.Contains(s.resultText(result), "cannot start with '-'")
}

func (s *gitServerTestSuite) TestGitDiff_MissingTarget() {
	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": s.repoDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiff_InvalidTarget() {
	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": s.repoDir, "target": "nonexistent-branch"})
	s.True(result.IsError)
}

// --- git_add ---

func (s *gitServerTestSuite) TestGitAdd_SpecificFile() {
	s.writeFile("add_test.txt", "content\n")
	result := s.callTool(ToolGitAdd, map[string]any{
		"repo_path": s.repoDir,
		"files":     []string{"add_test.txt"},
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "Files staged successfully")
}

func (s *gitServerTestSuite) TestGitAdd_AllFiles() {
	s.writeFile("file1.txt", "content1\n")
	s.writeFile("file2.txt", "content2\n")
	result := s.callTool(ToolGitAdd, map[string]any{
		"repo_path": s.repoDir,
		"files":     []string{"."},
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "Files staged successfully")
}

func (s *gitServerTestSuite) TestGitAdd_EmptyFiles() {
	result := s.callTool(ToolGitAdd, map[string]any{
		"repo_path": s.repoDir,
		"files":     []string{},
	})
	s.True(result.IsError)
}

// --- git_commit ---

func (s *gitServerTestSuite) TestGitCommit_Success() {
	s.writeFile("commit_test.txt", "content\n")
	s.stageFile("commit_test.txt")
	result := s.callTool(ToolGitCommit, map[string]any{
		"repo_path": s.repoDir,
		"message":   "test commit",
	})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "Changes committed successfully with hash")
}

func (s *gitServerTestSuite) TestGitCommit_EmptyMessage() {
	result := s.callTool(ToolGitCommit, map[string]any{
		"repo_path": s.repoDir,
		"message":   "",
	})
	s.True(result.IsError)
}

// --- git_reset ---

func (s *gitServerTestSuite) TestGitReset_Success() {
	s.writeFile("reset_test.txt", "content\n")
	s.stageFile("reset_test.txt")
	result := s.callTool(ToolGitReset, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "All staged changes reset")
}

// --- git_log ---

func (s *gitServerTestSuite) TestGitLog_Default() {
	result := s.callTool(ToolGitLog, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "commit ")
	s.Contains(text, "initial commit")
}

func (s *gitServerTestSuite) TestGitLog_MaxCount() {
	// Add more commits
	for i := 0; i < 3; i++ {
		s.writeFile("log_file.txt", strings.Repeat("x", i+1))
		s.stageFile("log_file.txt")
		s.commit("commit " + string(rune('0'+i)))
	}
	result := s.callTool(ToolGitLog, map[string]any{"repo_path": s.repoDir, "max_count": 2})
	s.False(result.IsError)
	text := s.resultText(result)
	// Should contain exactly 2 entries (each has an Author: line)
	count := strings.Count(text, "Author:")
	s.Equal(2, count)
}

func (s *gitServerTestSuite) TestGitLog_WithStartTimestamp() {
	result := s.callTool(ToolGitLog, map[string]any{
		"repo_path":       s.repoDir,
		"start_timestamp": "2020-01-01",
	})
	s.False(result.IsError)
}

func (s *gitServerTestSuite) TestGitLog_WithEndTimestamp() {
	result := s.callTool(ToolGitLog, map[string]any{
		"repo_path":     s.repoDir,
		"end_timestamp": "2099-12-31",
	})
	s.False(result.IsError)
}

func (s *gitServerTestSuite) TestGitLog_InvalidTimestamp() {
	result := s.callTool(ToolGitLog, map[string]any{
		"repo_path":       s.repoDir,
		"start_timestamp": "not-a-date",
	})
	s.True(result.IsError)
}

// --- git_create_branch ---

func (s *gitServerTestSuite) TestGitCreateBranch_FromHead() {
	result := s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "new-feature",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "new-feature")
}

func (s *gitServerTestSuite) TestGitCreateBranch_FromBase() {
	// Create base branch first
	s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "base-branch",
	})
	result := s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "derived-branch",
		"base_branch": "base-branch",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "derived-branch")
}

func (s *gitServerTestSuite) TestGitCreateBranch_InvalidBase() {
	result := s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "new-branch",
		"base_branch": "nonexistent-base",
	})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitCreateBranch_MissingName() {
	result := s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "",
	})
	s.True(result.IsError)
}

// --- git_checkout ---

func (s *gitServerTestSuite) TestGitCheckout_ExistingBranch() {
	s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "checkout-target",
	})
	result := s.callTool(ToolGitCheckout, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "checkout-target",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "Switched to branch")
}

func (s *gitServerTestSuite) TestGitCheckout_NonExistentBranch() {
	result := s.callTool(ToolGitCheckout, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "nonexistent-branch",
	})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitCheckout_FlagInjectionRejected() {
	result := s.callTool(ToolGitCheckout, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "--help",
	})
	s.True(result.IsError)
	s.Contains(s.resultText(result), "cannot start with '-'")
}

func (s *gitServerTestSuite) TestGitCheckout_MissingBranchName() {
	result := s.callTool(ToolGitCheckout, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "",
	})
	s.True(result.IsError)
}

// --- git_show ---

func (s *gitServerTestSuite) TestGitShow_LatestCommit() {
	head, err := s.repo.Head()
	s.Require().NoError(err)
	result := s.callTool(ToolGitShow, map[string]any{
		"repo_path": s.repoDir,
		"revision":  head.Hash().String(),
	})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "commit ")
	s.Contains(text, "Author:")
	s.Contains(text, "initial commit")
}

func (s *gitServerTestSuite) TestGitShow_HeadRevision() {
	result := s.callTool(ToolGitShow, map[string]any{
		"repo_path": s.repoDir,
		"revision":  "HEAD",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "commit ")
}

func (s *gitServerTestSuite) TestGitShow_SecondCommit() {
	s.writeFile("second.txt", "second file\n")
	s.stageFile("second.txt")
	hash := s.commit("second commit")
	result := s.callTool(ToolGitShow, map[string]any{
		"repo_path": s.repoDir,
		"revision":  hash,
	})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "second commit")
}

func (s *gitServerTestSuite) TestGitShow_InvalidRevision() {
	result := s.callTool(ToolGitShow, map[string]any{
		"repo_path": s.repoDir,
		"revision":  "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitShow_MissingRevision() {
	result := s.callTool(ToolGitShow, map[string]any{
		"repo_path": s.repoDir,
		"revision":  "",
	})
	s.True(result.IsError)
}

// --- git_branch ---

func (s *gitServerTestSuite) TestGitBranch_Local() {
	s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "local-branch",
	})
	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "local",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "local-branch")
}

func (s *gitServerTestSuite) TestGitBranch_Remote() {
	// No remotes configured — should return empty
	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "remote",
	})
	s.False(result.IsError)
	s.Empty(strings.TrimSpace(s.resultText(result)))
}

func (s *gitServerTestSuite) TestGitBranch_All() {
	s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_name": "all-branch",
	})
	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "all",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "all-branch")
}

func (s *gitServerTestSuite) TestGitBranch_Contains() {
	// Create a branch with a unique commit
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/contains-branch",
		Create: true,
	})
	s.Require().NoError(err)
	s.writeFile("unique.txt", "unique\n")
	s.stageFile("unique.txt")
	uniqueHash := s.commit("unique commit")

	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "local",
		"contains":    uniqueHash,
	})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "contains-branch")
	s.NotContains(text, "main")
}

func (s *gitServerTestSuite) TestGitBranch_NotContains() {
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/not-contains-branch",
		Create: true,
	})
	s.Require().NoError(err)
	s.writeFile("unique2.txt", "unique2\n")
	s.stageFile("unique2.txt")
	uniqueHash := s.commit("unique2 commit")

	// Go back to default branch
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(s.defaultBranch),
	})
	s.Require().NoError(err)

	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":    s.repoDir,
		"branch_type":  "local",
		"not_contains": uniqueHash,
	})
	s.False(result.IsError)
	text := s.resultText(result)
	s.NotContains(text, "not-contains-branch")
	s.Contains(text, s.defaultBranch)
}

func (s *gitServerTestSuite) TestGitBranch_InvalidType() {
	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "invalid",
	})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitBranch_MissingType() {
	result := s.callTool(ToolGitBranch, map[string]any{
		"repo_path":   s.repoDir,
		"branch_type": "",
	})
	s.True(result.IsError)
}

// --- repo restriction ---

func (s *gitServerTestSuite) TestRepoRestriction_Enforced() {
	// Create a git server restricted to a different directory
	restrictedDir, err := os.MkdirTemp("", "restricted-*")
	s.Require().NoError(err)
	s.Require().NoError(os.Mkdir(filepath.Join(restrictedDir, "allowed"), 0o755))
	defer os.RemoveAll(restrictedDir)

	allowedPath := filepath.Join(restrictedDir, "allowed")
	gsRestricted := &GitServer{options: Options{AllowedRepository: allowedPath}}

	argsBytes, _ := json.Marshal(map[string]any{"repo_path": s.repoDir})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      ToolGitStatus,
			Arguments: argsBytes,
		},
	}
	result, err := gsRestricted.handleGitStatus(context.Background(), req)
	s.Require().NoError(err)
	s.True(result.IsError)
	s.Contains(s.resultText(result), "outside the allowed repository")
}

// --- timestamp parsing ---

func (s *gitServerTestSuite) TestParseTimestamp_ISO8601() {
	t, err := parseTimestamp("2024-01-15T14:30:25Z")
	s.NoError(err)
	s.Equal(2024, t.Year())
}

func (s *gitServerTestSuite) TestParseTimestamp_DateOnly() {
	t, err := parseTimestamp("2024-01-15")
	s.NoError(err)
	s.Equal(2024, t.Year())
	s.Equal(time.January, t.Month())
}

func (s *gitServerTestSuite) TestParseTimestamp_RelativeWeeks() {
	t, err := parseTimestamp("2 weeks ago")
	s.NoError(err)
	expected := time.Now().AddDate(0, 0, -14)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_RelativeDays() {
	t, err := parseTimestamp("3 days ago")
	s.NoError(err)
	expected := time.Now().AddDate(0, 0, -3)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_Yesterday() {
	t, err := parseTimestamp("yesterday")
	s.NoError(err)
	expected := time.Now().AddDate(0, 0, -1)
	s.InDelta(expected.Unix(), t.Unix(), 60)
}

func (s *gitServerTestSuite) TestParseTimestamp_Invalid() {
	_, err := parseTimestamp("not-a-date")
	s.Error(err)
}

// --- relative time parsing coverage ---

func (s *gitServerTestSuite) TestParseTimestamp_RelativeHours() {
	t, err := parseTimestamp("2 hours ago")
	s.NoError(err)
	expected := time.Now().Add(-2 * time.Hour)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_RelativeMinutes() {
	t, err := parseTimestamp("30 minutes ago")
	s.NoError(err)
	expected := time.Now().Add(-30 * time.Minute)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_RelativeMonths() {
	t, err := parseTimestamp("3 months ago")
	s.NoError(err)
	expected := time.Now().AddDate(0, -3, 0)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_RelativeYears() {
	t, err := parseTimestamp("1 year ago")
	s.NoError(err)
	expected := time.Now().AddDate(-1, 0, 0)
	s.InDelta(expected.Unix(), t.Unix(), 5)
}

func (s *gitServerTestSuite) TestParseTimestamp_Today() {
	t, err := parseTimestamp("today")
	s.NoError(err)
	now := time.Now()
	s.Equal(now.Year(), t.Year())
	s.Equal(now.Month(), t.Month())
	s.Equal(now.Day(), t.Day())
}

func (s *gitServerTestSuite) TestParseTimestamp_UnrecognizedUnit() {
	_, err := parseTimestamp("5 fortnights ago")
	s.Error(err)
}

func (s *gitServerTestSuite) TestParseTimestamp_NoUnit() {
	// Sscanf won't match — falls to unrecognized relative time error
	_, err := parseTimestamp("foobar")
	s.Error(err)
}

// --- git_diff_staged with modified existing file ---

func (s *gitServerTestSuite) TestGitDiffStaged_ModifiedExistingFile() {
	// Modify and stage an already-tracked file
	s.writeFile("readme.txt", "modified existing content\n")
	s.stageFile("readme.txt")
	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "readme.txt")
}

// --- git_log zero max_count ---

func (s *gitServerTestSuite) TestGitLog_ZeroMaxCount() {
	// max_count=0 means unlimited — should return all commits
	result := s.callTool(ToolGitLog, map[string]any{"repo_path": s.repoDir, "max_count": 0})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "initial commit")
}

// --- bare repo to cover worktree error paths ---

func (s *gitServerTestSuite) TestGitStatus_BareRepo() {
	// A bare repo has no worktree — gitStatus's Worktree() call will error.
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": bareDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_BareRepo() {
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": bareDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiffStaged_BareRepo() {
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": bareDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitCommit_BareRepo() {
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitCommit, map[string]any{"repo_path": bareDir, "message": "test"})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitAdd_BareRepo() {
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitAdd, map[string]any{"repo_path": bareDir, "files": []string{"file.txt"}})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitReset_BareRepo() {
	bareDir, err := os.MkdirTemp("", "git-bare-*")
	s.Require().NoError(err)
	defer os.RemoveAll(bareDir)
	_, err = gogit.PlainInit(bareDir, true)
	s.Require().NoError(err)

	result := s.callTool(ToolGitReset, map[string]any{"repo_path": bareDir})
	s.True(result.IsError)
}

// --- empty repo (no commits) to cover HEAD error paths ---

func (s *gitServerTestSuite) TestGitDiffUnstaged_EmptyRepo() {
	// A repo with no commits has no HEAD — exercises the "getting HEAD" error path
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": emptyDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiffStaged_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": emptyDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiff_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": emptyDir, "target": "main"})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitReset_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitReset, map[string]any{"repo_path": emptyDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitLog_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitLog, map[string]any{"repo_path": emptyDir})
	s.True(result.IsError)
}

func (s *gitServerTestSuite) TestGitShow_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitShow, map[string]any{"repo_path": emptyDir, "revision": "HEAD"})
	s.True(result.IsError)
}

// --- gitCommit error: nothing to commit ---

func (s *gitServerTestSuite) TestGitCommit_NothingToCommit() {
	// Committing with nothing staged should return an error from go-git
	result := s.callTool(ToolGitCommit, map[string]any{
		"repo_path": s.repoDir,
		"message":   "empty commit attempt",
	})
	s.True(result.IsError)
}

// --- gitAdd error: staging non-existent file ---

func (s *gitServerTestSuite) TestGitAdd_NonExistentFile() {
	result := s.callTool(ToolGitAdd, map[string]any{
		"repo_path": s.repoDir,
		"files":     []string{"non_existent_file.txt"},
	})
	s.True(result.IsError)
}

// --- gitCreateBranch on empty repo ---

func (s *gitServerTestSuite) TestGitCreateBranch_EmptyRepo() {
	emptyDir, err := os.MkdirTemp("", "git-empty-*")
	s.Require().NoError(err)
	defer os.RemoveAll(emptyDir)
	_, err = gogit.PlainInit(emptyDir, false)
	s.Require().NoError(err)

	result := s.callTool(ToolGitCreateBranch, map[string]any{
		"repo_path":   emptyDir,
		"branch_name": "new-branch",
	})
	s.True(result.IsError)
}

// Coverage note: Some remaining error branches require simulating specific
// mid-operation I/O failures or corrupt git object stores (e.g. gitShow's
// parent/tree/patch computation failures, encodePatch writer errors,
// gitCheckout's wt error after ResolveRevision succeeds).
// These cannot be triggered without mocking go-git internals and are documented
// here per AGENTS.md exception policy.

// --- JSON decode error paths for handlers ---
// These cover the json.Unmarshal branches that aren't reachable with valid args.

func (s *gitServerTestSuite) TestHandlers_InvalidJSON() {
	ctx := context.Background()
	badJSON := []byte(`{not valid json`)

	handlers := []struct {
		name    string
		handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{ToolGitStatus, s.gs.handleGitStatus},
		{ToolGitDiffUnstaged, s.gs.handleGitDiffUnstaged},
		{ToolGitDiffStaged, s.gs.handleGitDiffStaged},
		{ToolGitDiff, s.gs.handleGitDiff},
		{ToolGitCommit, s.gs.handleGitCommit},
		{ToolGitAdd, s.gs.handleGitAdd},
		{ToolGitReset, s.gs.handleGitReset},
		{ToolGitLog, s.gs.handleGitLog},
		{ToolGitCreateBranch, s.gs.handleGitCreateBranch},
		{ToolGitCheckout, s.gs.handleGitCheckout},
		{ToolGitShow, s.gs.handleGitShow},
		{ToolGitBranch, s.gs.handleGitBranch},
	}

	for _, h := range handlers {
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{Name: h.name, Arguments: badJSON},
		}
		result, err := h.handler(ctx, req)
		s.NoError(err, "handler %s should not return Go error", h.name)
		s.True(result.IsError, "handler %s should return IsError=true for bad JSON", h.name)
	}
}

// --- git_log with relative timestamp ---

func (s *gitServerTestSuite) TestGitLog_RelativeStartTimestamp() {
	result := s.callTool(ToolGitLog, map[string]any{
		"repo_path":       s.repoDir,
		"start_timestamp": "1 year ago",
	})
	s.False(result.IsError)
	s.Contains(s.resultText(result), "initial commit")
}

// --- JSON output format ---

func (s *gitServerTestSuite) TestGitStatus_JSONFormat() {
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	text := s.resultText(result)
	var parsed StatusResult
	s.NoError(json.Unmarshal([]byte(text), &parsed))
	s.Equal(s.defaultBranch, parsed.Repository.Branch)
	s.NotEmpty(parsed.Repository.HeadSHA)
	s.Equal(0, parsed.Summary.TotalFiles)
}

func (s *gitServerTestSuite) TestGitStatus_JSONFormat_Dirty() {
	s.writeFile("new_file.txt", "content")
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	var parsed StatusResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal(1, parsed.Summary.UntrackedCount)
}

func (s *gitServerTestSuite) TestGitLog_JSONFormat() {
	result := s.callTool(ToolGitLog, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	var parsed LogResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Len(parsed.Commits, 1)
	s.Equal("initial commit", parsed.Commits[0].Message)
	s.NotEmpty(parsed.Commits[0].Author.Email)
	s.NotEmpty(parsed.Commits[0].Refs)
}

func (s *gitServerTestSuite) TestGitShow_JSONFormat() {
	result := s.callTool(ToolGitShow, map[string]any{"repo_path": s.repoDir, "revision": "HEAD", "format": "json"})
	s.False(result.IsError)
	var parsed ShowResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("initial commit", parsed.Commit.Message)
	s.NotEmpty(parsed.Diff.Files)
}

func (s *gitServerTestSuite) TestGitBranch_JSONFormat() {
	result := s.callTool(ToolGitBranch, map[string]any{"repo_path": s.repoDir, "branch_type": "local", "format": "json"})
	s.False(result.IsError)
	var parsed BranchResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal(s.defaultBranch, parsed.CurrentBranch)
	s.Len(parsed.Branches, 1)
	s.True(parsed.Branches[0].IsCurrent)
	s.NotEmpty(parsed.Branches[0].LastCommitSHA)
	s.NotEmpty(parsed.Branches[0].LastCommitDate)
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_JSONFormat() {
	s.writeFile("readme.txt", "modified content\n")
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	var parsed DiffResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("has_changes", parsed.Status)
	s.NotEmpty(parsed.Files)
	s.Greater(parsed.Summary.TotalAdditions+parsed.Summary.TotalDeletions, 0)
}

func (s *gitServerTestSuite) TestGitDiffStaged_JSONFormat() {
	s.writeFile("staged.txt", "staged content\n")
	s.stageFile("staged.txt")
	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	var parsed DiffResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("has_changes", parsed.Status)
	s.NotEmpty(parsed.Files)
}

func (s *gitServerTestSuite) TestGitDiff_JSONFormat() {
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	err = wt.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/json-feature", Create: true})
	s.Require().NoError(err)
	s.writeFile("json_feature.txt", "json feature\n")
	s.stageFile("json_feature.txt")
	s.commit("json feature commit")

	result := s.callTool(ToolGitDiff, map[string]any{"repo_path": s.repoDir, "target": s.defaultBranch, "format": "json"})
	s.False(result.IsError)
	var parsed DiffResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("has_changes", parsed.Status)
	s.Equal(s.defaultBranch, parsed.Base)
	s.Equal("HEAD", parsed.Target)
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_JSONFormat_Empty() {
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir, "format": "json"})
	s.False(result.IsError)
	var parsed DiffResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("no_changes", parsed.Status)
	s.Empty(parsed.Files)
}

// --- resolveFormat ---

func (s *gitServerTestSuite) TestResolveFormat_PerCallOverride() {
	gs := &GitServer{options: Options{OutputFormat: FormatText}}
	s.Equal(FormatJSON, gs.resolveFormat("json"))
	s.Equal(FormatText, gs.resolveFormat("text"))
}

func (s *gitServerTestSuite) TestResolveFormat_ServerDefault() {
	gs := &GitServer{options: Options{OutputFormat: FormatJSON}}
	s.Equal(FormatJSON, gs.resolveFormat(""))
	s.Equal(FormatJSON, gs.resolveFormat("bogus"))
}

func (s *gitServerTestSuite) TestResolveFormat_DefaultsToText() {
	gs := &GitServer{options: Options{}}
	s.Equal(FormatText, gs.resolveFormat(""))
}

func (s *gitServerTestSuite) TestResolveFormat_ServerDefaultText() {
	gs := &GitServer{options: Options{OutputFormat: FormatText}}
	s.Equal(FormatText, gs.resolveFormat(""))
}

// --- formatter unit tests ---

func (s *gitServerTestSuite) TestFormatLogText_Empty() {
	result := formatLogText(&LogResult{Commits: []CommitInfo{}})
	s.Empty(result)
}

func (s *gitServerTestSuite) TestFormatLogText_MergeCommit() {
	result := formatLogText(&LogResult{Commits: []CommitInfo{
		{
			SHA:     "abc123",
			Author:  AuthorInfo{Name: "Test", Email: "t@t.com"},
			Date:    "2024-01-01T00:00:00Z",
			Message: "merge",
			Parents: []string{"parent1abcdef0123456789abcdef0123456789ab", "parent2"},
			Refs:    []string{},
		},
	}})
	s.Contains(result, "Merge: parent1 parent2")
}

func (s *gitServerTestSuite) TestFormatBranchText_Mixed() {
	result := formatBranchText(&BranchResult{
		CurrentBranch: "main",
		Branches: []BranchInfo{
			{Name: "feature", IsCurrent: false},
			{Name: "main", IsCurrent: true},
		},
	})
	s.Contains(result, "* main")
	s.Contains(result, "  feature")
}

func (s *gitServerTestSuite) TestFormatStatusText_WithRemote() {
	result := formatStatusText(&StatusResult{
		Repository: RepositoryInfo{
			Branch:  "main",
			HeadSHA: "abc",
			Remote: &RemoteInfo{
				Name:    "origin",
				Branch:  "origin/main",
				Status:  "ahead",
				AheadBy: 2,
			},
		},
		Changes: StatusChanges{
			Staged:    []FileChange{{Path: "file.go", Status: "added"}},
			Unstaged:  []FileChange{{Path: "other.go", Status: "modified"}},
			Untracked: []FileChange{{Path: "new.txt", Status: "untracked"}},
			Conflicts: []FileChange{{Path: "conflict.go", Status: "conflict"}},
		},
		Summary: StatusSummary{TotalFiles: 4, StagedCount: 1, UnstagedCount: 1, UntrackedCount: 1, ConflictedCount: 1},
	})
	s.Contains(result, "Your branch is ahead")
	s.Contains(result, "Changes to be committed:")
	s.Contains(result, "new file:   file.go")
	s.Contains(result, "Changes not staged for commit:")
	s.Contains(result, "modified:   other.go")
	s.Contains(result, "Untracked files:")
	s.Contains(result, "new.txt")
	s.Contains(result, "Unmerged paths:")
	s.Contains(result, "conflict.go")
}

func (s *gitServerTestSuite) TestFormatStatusText_RemoteBehind() {
	result := formatStatusText(&StatusResult{
		Repository: RepositoryInfo{
			Branch: "main",
			Remote: &RemoteInfo{Branch: "origin/main", Status: "behind", BehindBy: 3},
		},
		Changes: StatusChanges{Staged: []FileChange{}, Unstaged: []FileChange{}, Untracked: []FileChange{}, Conflicts: []FileChange{}},
	})
	s.Contains(result, "Your branch is behind")
}

func (s *gitServerTestSuite) TestFormatStatusText_RemoteDiverged() {
	result := formatStatusText(&StatusResult{
		Repository: RepositoryInfo{
			Branch: "main",
			Remote: &RemoteInfo{Branch: "origin/main", Status: "diverged"},
		},
		Changes: StatusChanges{Staged: []FileChange{}, Unstaged: []FileChange{}, Untracked: []FileChange{}, Conflicts: []FileChange{}},
	})
	s.Contains(result, "have diverged")
}

func (s *gitServerTestSuite) TestFormatDiffText_NoChanges() {
	result := formatDiffText(&DiffResult{Status: "no_changes"})
	s.Empty(result)
}

func (s *gitServerTestSuite) TestFormatDiffText_UsesRawText() {
	result := formatDiffText(&DiffResult{RawText: "raw diff output", Status: "has_changes"})
	s.Equal("raw diff output", result)
}

func (s *gitServerTestSuite) TestFormatDiffText_RendersFromStruct() {
	result := formatDiffText(&DiffResult{
		Status: "has_changes",
		Files: []DiffFile{{
			Path:   "file.go",
			Status: "modified",
			Changes: []DiffChange{
				{Type: "hunk_header", NewContent: "@@ -1,3 +1,3 @@"},
				{Type: "context", OldContent: "unchanged"},
				{Type: "deletion", OldContent: "old line"},
				{Type: "addition", NewContent: "new line"},
			},
		}},
	})
	s.Contains(result, "diff --git a/file.go b/file.go")
	s.Contains(result, "--- a/file.go")
	s.Contains(result, "+++ b/file.go")
	s.Contains(result, " unchanged")
	s.Contains(result, "-old line")
	s.Contains(result, "+new line")
}

func (s *gitServerTestSuite) TestFormatDiffText_AddedFile() {
	result := formatDiffText(&DiffResult{
		Status: "has_changes",
		Files: []DiffFile{{
			Path:    "new.go",
			Status:  "added",
			Changes: []DiffChange{{Type: "addition", NewContent: "package main"}},
		}},
	})
	s.Contains(result, "new file mode")
	s.Contains(result, "--- /dev/null")
}

func (s *gitServerTestSuite) TestFormatDiffText_DeletedFile() {
	result := formatDiffText(&DiffResult{
		Status: "has_changes",
		Files: []DiffFile{{
			Path:    "old.go",
			Status:  "deleted",
			Changes: []DiffChange{{Type: "deletion", OldContent: "package main"}},
		}},
	})
	s.Contains(result, "deleted file mode")
	s.Contains(result, "+++ /dev/null")
}

func (s *gitServerTestSuite) TestFormatDiffText_BinaryFile() {
	result := formatDiffText(&DiffResult{
		Status: "has_changes",
		Files: []DiffFile{{
			Path:   "image.png",
			Status: "modified",
			Binary: true,
		}},
	})
	s.Contains(result, "Binary files")
}

func (s *gitServerTestSuite) TestFormatDiffText_RenamedFile() {
	result := formatDiffText(&DiffResult{
		Status: "has_changes",
		Files: []DiffFile{{
			Path:    "new_name.go",
			OldPath: "old_name.go",
			Status:  "renamed",
			Changes: []DiffChange{},
		}},
	})
	s.Contains(result, "diff --git a/old_name.go b/new_name.go")
}

func (s *gitServerTestSuite) TestFormatShowText_MergeCommit() {
	result := formatShowText(&ShowResult{
		Commit: CommitInfo{
			SHA:     "abc123",
			Author:  AuthorInfo{Name: "Test", Email: "t@t.com"},
			Date:    "2024-01-01T00:00:00Z",
			Message: "merge commit",
			Parents: []string{"parent1abcdef0123456789abcdef0123456789ab", "short"},
			Refs:    []string{"HEAD -> main"},
		},
		Diff: DiffResult{Status: "no_changes"},
	})
	s.Contains(result, "commit abc123 (HEAD -> main)")
	s.Contains(result, "Merge: parent1 short")
}

func (s *gitServerTestSuite) TestFormatErrorJSON() {
	result := formatErrorJSON(fmt.Errorf("something broke"))
	s.Contains(result, "something broke")
	s.Contains(result, "\"code\":\"error\"")
}

func (s *gitServerTestSuite) TestStatusCodeToText_AllBranches() {
	s.Equal("new file:   ", statusCodeToText("added"))
	s.Equal("modified:   ", statusCodeToText("modified"))
	s.Equal("deleted:    ", statusCodeToText("deleted"))
	s.Equal("renamed:    ", statusCodeToText("renamed"))
	s.Equal("copied:     ", statusCodeToText("copied"))
	s.Equal("unknown: ", statusCodeToText("unknown"))
}

// --- helper function unit tests ---

func (s *gitServerTestSuite) TestSplitChunkLines_Empty() {
	s.Nil(splitChunkLines(""))
}

func (s *gitServerTestSuite) TestSplitChunkLines_WithTrailingNewline() {
	result := splitChunkLines("a\nb\n")
	s.Equal([]string{"a", "b"}, result)
}

func (s *gitServerTestSuite) TestSplitChunkLines_NoTrailingNewline() {
	result := splitChunkLines("a\nb")
	s.Equal([]string{"a", "b"}, result)
}

func (s *gitServerTestSuite) TestManualDiffToFile_SameContent() {
	result := manualDiffToFile("file.go", "same", "same", "modified", 3)
	s.Nil(result)
}

func (s *gitServerTestSuite) TestManualDiffToFile_WithChanges() {
	result := manualDiffToFile("file.go", "old\n", "new\n", "modified", 3)
	s.NotNil(result)
	s.Equal("file.go", result.Path)
	s.Greater(result.Additions, 0)
	s.Greater(result.Deletions, 0)
}

func (s *gitServerTestSuite) TestStatusCodeToString_AllCodes() {
	s.Equal("added", statusCodeToString(gogit.Added))
	s.Equal("modified", statusCodeToString(gogit.Modified))
	s.Equal("deleted", statusCodeToString(gogit.Deleted))
	s.Equal("renamed", statusCodeToString(gogit.Renamed))
	s.Equal("copied", statusCodeToString(gogit.Copied))
	s.Equal("modified", statusCodeToString(gogit.UpdatedButUnmerged))
}

// --- server default format ---

func (s *gitServerTestSuite) TestGitLog_ServerDefaultJSON() {
	// Create a server with JSON as default
	srv, err := NewServer(context.Background(), Options{OutputFormat: FormatJSON})
	s.Require().NoError(err)
	gsJSON := &GitServer{server: srv, options: Options{OutputFormat: FormatJSON}}

	argsBytes, _ := json.Marshal(map[string]any{"repo_path": s.repoDir})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: ToolGitLog, Arguments: argsBytes}}
	result, err := gsJSON.handleGitLog(context.Background(), req)
	s.Require().NoError(err)
	text := result.Content[0].(*mcp.TextContent).Text
	var parsed LogResult
	s.NoError(json.Unmarshal([]byte(text), &parsed))
	s.NotEmpty(parsed.Commits)
}

func (s *gitServerTestSuite) TestGitStatus_ServerDefaultJSON_PerCallOverrideText() {
	srv, err := NewServer(context.Background(), Options{OutputFormat: FormatJSON})
	s.Require().NoError(err)
	gsJSON := &GitServer{server: srv, options: Options{OutputFormat: FormatJSON}}

	argsBytes, _ := json.Marshal(map[string]any{"repo_path": s.repoDir, "format": "text"})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: ToolGitStatus, Arguments: argsBytes}}
	result, err := gsJSON.handleGitStatus(context.Background(), req)
	s.Require().NoError(err)
	text := result.Content[0].(*mcp.TextContent).Text
	s.Contains(text, "On branch")
}

// --- git_show with second commit for parent diff coverage ---

func (s *gitServerTestSuite) TestGitShow_SecondCommit_JSON() {
	s.writeFile("second.txt", "second file\n")
	s.stageFile("second.txt")
	hash := s.commit("second commit")
	result := s.callTool(ToolGitShow, map[string]any{"repo_path": s.repoDir, "revision": hash, "format": "json"})
	s.False(result.IsError)
	var parsed ShowResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.Equal("second commit", parsed.Commit.Message)
	s.NotEmpty(parsed.Commit.Parents)
}

// --- git_status with staged file for full coverage ---

func (s *gitServerTestSuite) TestGitStatus_StagedFile() {
	s.writeFile("staged.txt", "content")
	s.stageFile("staged.txt")
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "Changes to be committed")
	s.Contains(text, "staged.txt")
}

func (s *gitServerTestSuite) TestGitStatus_ModifiedFile() {
	s.writeFile("readme.txt", "changed content")
	result := s.callTool(ToolGitStatus, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "Changes not staged for commit")
	s.Contains(text, "readme.txt")
}

func (s *gitServerTestSuite) TestSortFileChanges() {
	changes := []FileChange{
		{Path: "z.go", Status: "modified"},
		{Path: "a.go", Status: "added"},
		{Path: "m.go", Status: "deleted"},
	}
	sortFileChanges(changes)
	s.Equal("a.go", changes[0].Path)
	s.Equal("m.go", changes[1].Path)
	s.Equal("z.go", changes[2].Path)
}

func (s *gitServerTestSuite) TestFormatStatusText_RemoteUpToDate() {
	result := formatStatusText(&StatusResult{
		Repository: RepositoryInfo{
			Branch: "main",
			Remote: &RemoteInfo{Branch: "origin/main", Status: "up_to_date"},
		},
		Changes: StatusChanges{},
	})
	s.Contains(result, "up to date")
}

func (s *gitServerTestSuite) TestGitDiffUnstaged_DeletedFile() {
	// Deleted worktree files can't be opened, so gitDiffUnstaged skips them.
	// This exercises the wt.Filesystem.Open error path (continue on line 422).
	os.Remove(filepath.Join(s.repoDir, "readme.txt"))
	result := s.callTool(ToolGitDiffUnstaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
}

func (s *gitServerTestSuite) TestGitDiffStaged_DeletedFile() {
	wt, err := s.repo.Worktree()
	s.Require().NoError(err)
	os.Remove(filepath.Join(s.repoDir, "readme.txt"))
	_, err = wt.Add("readme.txt")
	s.Require().NoError(err)
	result := s.callTool(ToolGitDiffStaged, map[string]any{"repo_path": s.repoDir})
	s.False(result.IsError)
	text := s.resultText(result)
	s.Contains(text, "readme.txt")
}

func (s *gitServerTestSuite) TestGitShow_InitialCommit_JSON_DiffFiles() {
	result := s.callTool(ToolGitShow, map[string]any{"repo_path": s.repoDir, "revision": "HEAD", "format": "json"})
	s.False(result.IsError)
	var parsed ShowResult
	s.NoError(json.Unmarshal([]byte(s.resultText(result)), &parsed))
	s.NotEmpty(parsed.Diff.Files)
	s.Equal("added", parsed.Diff.Files[0].Status)
	s.Greater(parsed.Diff.Summary.TotalAdditions, 0)
}

func TestGitServerSuite(t *testing.T) {
	suite.Run(t, new(gitServerTestSuite))
}

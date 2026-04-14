package git

const (
	DefaultContextLines = 3

	ToolGitStatus       = "git_status"
	ToolGitDiffUnstaged = "git_diff_unstaged"
	ToolGitDiffStaged   = "git_diff_staged"
	ToolGitDiff         = "git_diff"
	ToolGitCommit       = "git_commit"
	ToolGitAdd          = "git_add"
	ToolGitReset        = "git_reset"
	ToolGitLog          = "git_log"
	ToolGitCreateBranch = "git_create_branch"
	ToolGitCheckout     = "git_checkout"
	ToolGitShow         = "git_show"
	ToolGitBranch       = "git_branch"
)

// Options holds server-level configuration.
type Options struct {
	// AllowedRepository restricts all operations to repos within this path.
	// Empty string means no restriction.
	AllowedRepository string
}

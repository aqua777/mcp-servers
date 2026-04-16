package git

const (
	DefaultContextLines = 3

	FormatText = "text"
	FormatJSON = "json"

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

	// OutputFormat sets the default output format for read operations.
	// Valid values: "text" (default), "json".
	// Can be overridden per-call via the "format" tool parameter.
	OutputFormat string

	// AIMode enables AI-first defaults: JSON output, structured errors.
	// When true, defaults to JSON format unless explicitly overridden.
	AIMode bool
}

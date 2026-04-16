package git

// AuthorInfo represents commit author metadata.
type AuthorInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CommitInfo represents a single commit's metadata.
type CommitInfo struct {
	SHA     string     `json:"sha"`
	Author  AuthorInfo `json:"author"`
	Date    string     `json:"date"`
	Message string     `json:"message"`
	Refs    []string   `json:"refs"`
	Parents []string   `json:"parents"`
}

// LogResult holds the structured result of a git log operation.
type LogResult struct {
	Commits []CommitInfo `json:"commits"`
}

// BranchInfo represents a single branch entry.
type BranchInfo struct {
	Name           string `json:"name"`
	IsCurrent      bool   `json:"is_current"`
	Tracking       string `json:"tracking,omitempty"`
	LastCommitSHA  string `json:"last_commit_sha,omitempty"`
	LastCommitDate string `json:"last_commit_date,omitempty"`
}

// BranchResult holds the structured result of a git branch operation.
type BranchResult struct {
	CurrentBranch string       `json:"current_branch"`
	IsDetached    bool         `json:"is_detached"`
	Branches      []BranchInfo `json:"branches"`
}

// FileChange represents a changed file in status or diff.
type FileChange struct {
	Path    string `json:"path"`
	OldPath string `json:"old_path,omitempty"`
	Status  string `json:"status"`
	Type    string `json:"type,omitempty"`
}

// RemoteInfo represents the remote tracking state.
type RemoteInfo struct {
	Name     string `json:"name,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Status   string `json:"status,omitempty"`
	AheadBy  int    `json:"ahead_by"`
	BehindBy int    `json:"behind_by"`
}

// RepositoryInfo represents current repository metadata.
type RepositoryInfo struct {
	Branch  string      `json:"branch"`
	HeadSHA string      `json:"head_sha"`
	Remote  *RemoteInfo `json:"remote,omitempty"`
}

// StatusChanges holds categorized file changes.
type StatusChanges struct {
	Staged    []FileChange `json:"staged"`
	Unstaged  []FileChange `json:"unstaged"`
	Untracked []FileChange `json:"untracked"`
	Conflicts []FileChange `json:"conflicts"`
}

// StatusSummary provides aggregate counts.
type StatusSummary struct {
	TotalFiles      int `json:"total_files"`
	StagedCount     int `json:"staged_count"`
	UnstagedCount   int `json:"unstaged_count"`
	UntrackedCount  int `json:"untracked_count"`
	ConflictedCount int `json:"conflicted_count"`
}

// StatusResult holds the structured result of a git status operation.
type StatusResult struct {
	Repository RepositoryInfo `json:"repository"`
	Changes    StatusChanges  `json:"changes"`
	Summary    StatusSummary  `json:"summary"`
}

// DiffChange represents a single line-level change within a file.
type DiffChange struct {
	Type          string   `json:"type"`
	OldLine       int      `json:"old_line,omitempty"`
	NewLine       int      `json:"new_line,omitempty"`
	OldContent    string   `json:"old_content,omitempty"`
	NewContent    string   `json:"new_content,omitempty"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
}

// DiffFile represents changes to a single file.
type DiffFile struct {
	Path      string       `json:"path"`
	OldPath   string       `json:"old_path,omitempty"`
	Status    string       `json:"status"`
	Binary    bool         `json:"binary"`
	Additions int          `json:"additions"`
	Deletions int          `json:"deletions"`
	Changes   []DiffChange `json:"changes"`
}

// DiffSummary provides aggregate diff statistics.
type DiffSummary struct {
	TotalFiles     int `json:"total_files"`
	TotalAdditions int `json:"total_additions"`
	TotalDeletions int `json:"total_deletions"`
}

// DiffResult holds the structured result of a diff operation.
type DiffResult struct {
	Status    string      `json:"status"`
	Base      string      `json:"base,omitempty"`
	Target    string      `json:"target,omitempty"`
	Files     []DiffFile  `json:"files"`
	Summary   DiffSummary `json:"summary"`
	Truncated bool        `json:"truncated,omitempty"`
	RawText   string      `json:"-"`
}

// ShowResult holds the structured result of a git show operation.
type ShowResult struct {
	Commit CommitInfo `json:"commit"`
	Diff   DiffResult `json:"diff"`
}

// ErrorResponse provides structured error output in JSON mode.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail holds error metadata.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

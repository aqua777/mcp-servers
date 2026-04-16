package git

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatLogText renders a LogResult in Git CLI format.
func formatLogText(r *LogResult) string {
	var sb strings.Builder
	for i, c := range r.Commits {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("commit ")
		sb.WriteString(c.SHA)
		if len(c.Refs) > 0 {
			sb.WriteString(" (")
			sb.WriteString(strings.Join(c.Refs, ", "))
			sb.WriteString(")")
		}
		sb.WriteString("\n")
		if len(c.Parents) > 1 {
			sb.WriteString("Merge: ")
			var short []string
			for _, p := range c.Parents {
				if len(p) > 7 {
					short = append(short, p[:7])
				} else {
					short = append(short, p)
				}
			}
			sb.WriteString(strings.Join(short, " "))
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Author: %s <%s>\n", c.Author.Name, c.Author.Email))
		sb.WriteString(fmt.Sprintf("Date:   %s\n", c.Date))
		sb.WriteString("\n")
		for _, line := range strings.Split(c.Message, "\n") {
			sb.WriteString("    ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// formatLogJSON renders a LogResult as JSON.
func formatLogJSON(r *LogResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatBranchText renders a BranchResult in Git CLI format.
func formatBranchText(r *BranchResult) string {
	var sb strings.Builder
	for i, br := range r.Branches {
		if i > 0 {
			sb.WriteString("\n")
		}
		if br.IsCurrent {
			sb.WriteString("* ")
		} else {
			sb.WriteString("  ")
		}
		sb.WriteString(br.Name)
	}
	return sb.String()
}

// formatBranchJSON renders a BranchResult as JSON.
func formatBranchJSON(r *BranchResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// statusCodeToText maps go-git status codes to human-readable strings.
func statusCodeToText(code string) string {
	switch code {
	case "added":
		return "new file:   "
	case "modified":
		return "modified:   "
	case "deleted":
		return "deleted:    "
	case "renamed":
		return "renamed:    "
	case "copied":
		return "copied:     "
	default:
		return code + ": "
	}
}

// formatStatusText renders a StatusResult matching git status CLI output.
func formatStatusText(r *StatusResult) string {
	var sb strings.Builder
	sb.WriteString("On branch ")
	sb.WriteString(r.Repository.Branch)
	sb.WriteString("\n")

	if r.Repository.Remote != nil && r.Repository.Remote.Branch != "" {
		switch r.Repository.Remote.Status {
		case "up_to_date":
			sb.WriteString(fmt.Sprintf("Your branch is up to date with '%s'.\n", r.Repository.Remote.Branch))
		case "ahead":
			sb.WriteString(fmt.Sprintf("Your branch is ahead of '%s' by %d commit(s).\n", r.Repository.Remote.Branch, r.Repository.Remote.AheadBy))
		case "behind":
			sb.WriteString(fmt.Sprintf("Your branch is behind '%s' by %d commit(s).\n", r.Repository.Remote.Branch, r.Repository.Remote.BehindBy))
		case "diverged":
			sb.WriteString(fmt.Sprintf("Your branch and '%s' have diverged.\n", r.Repository.Remote.Branch))
		}
	}

	if len(r.Changes.Staged) > 0 {
		sb.WriteString("\nChanges to be committed:\n")
		for _, f := range r.Changes.Staged {
			sb.WriteString("\t")
			sb.WriteString(statusCodeToText(f.Status))
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		}
	}

	if len(r.Changes.Unstaged) > 0 {
		sb.WriteString("\nChanges not staged for commit:\n")
		for _, f := range r.Changes.Unstaged {
			sb.WriteString("\t")
			sb.WriteString(statusCodeToText(f.Status))
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		}
	}

	if len(r.Changes.Untracked) > 0 {
		sb.WriteString("\nUntracked files:\n")
		for _, f := range r.Changes.Untracked {
			sb.WriteString("\t")
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		}
	}

	if len(r.Changes.Conflicts) > 0 {
		sb.WriteString("\nUnmerged paths:\n")
		for _, f := range r.Changes.Conflicts {
			sb.WriteString("\t")
			sb.WriteString(f.Path)
			sb.WriteString("\n")
		}
	}

	if r.Summary.TotalFiles == 0 {
		sb.WriteString("\nnothing to commit, working tree clean\n")
	}

	return sb.String()
}

// formatStatusJSON renders a StatusResult as JSON.
func formatStatusJSON(r *StatusResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatDiffText renders a DiffResult as unified diff text.
func formatDiffText(r *DiffResult) string {
	if r.RawText != "" {
		return r.RawText
	}
	if r.Status == "no_changes" {
		return ""
	}
	var sb strings.Builder
	for _, f := range r.Files {
		oldPath := f.OldPath
		if oldPath == "" {
			oldPath = f.Path
		}
		sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", oldPath, f.Path))
		if f.Binary {
			sb.WriteString(fmt.Sprintf("Binary files a/%s and b/%s differ\n", oldPath, f.Path))
			continue
		}
		switch f.Status {
		case "added":
			sb.WriteString("new file mode 100644\n")
			sb.WriteString("--- /dev/null\n")
			sb.WriteString(fmt.Sprintf("+++ b/%s\n", f.Path))
		case "deleted":
			sb.WriteString("deleted file mode 100644\n")
			sb.WriteString(fmt.Sprintf("--- a/%s\n", oldPath))
			sb.WriteString("+++ /dev/null\n")
		default:
			sb.WriteString(fmt.Sprintf("--- a/%s\n", oldPath))
			sb.WriteString(fmt.Sprintf("+++ b/%s\n", f.Path))
		}
		for _, ch := range f.Changes {
			switch ch.Type {
			case "hunk_header":
				sb.WriteString(ch.NewContent)
				sb.WriteString("\n")
			case "context":
				sb.WriteString(" ")
				sb.WriteString(ch.OldContent)
				sb.WriteString("\n")
			case "addition":
				sb.WriteString("+")
				sb.WriteString(ch.NewContent)
				sb.WriteString("\n")
			case "deletion":
				sb.WriteString("-")
				sb.WriteString(ch.OldContent)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// formatDiffJSON renders a DiffResult as JSON.
func formatDiffJSON(r *DiffResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatShowText renders a ShowResult in Git CLI format.
func formatShowText(r *ShowResult) string {
	var sb strings.Builder
	c := r.Commit
	sb.WriteString("commit ")
	sb.WriteString(c.SHA)
	if len(c.Refs) > 0 {
		sb.WriteString(" (")
		sb.WriteString(strings.Join(c.Refs, ", "))
		sb.WriteString(")")
	}
	sb.WriteString("\n")
	if len(c.Parents) > 1 {
		sb.WriteString("Merge: ")
		var short []string
		for _, p := range c.Parents {
			if len(p) > 7 {
				short = append(short, p[:7])
			} else {
				short = append(short, p)
			}
		}
		sb.WriteString(strings.Join(short, " "))
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Author: %s <%s>\n", c.Author.Name, c.Author.Email))
	sb.WriteString(fmt.Sprintf("Date:   %s\n", c.Date))
	sb.WriteString("\n")
	for _, line := range strings.Split(c.Message, "\n") {
		sb.WriteString("    ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(formatDiffText(&r.Diff))
	return sb.String()
}

// formatShowJSON renders a ShowResult as JSON.
func formatShowJSON(r *ShowResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

// formatErrorJSON renders an error as structured JSON.
func formatErrorJSON(err error) string {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    "error",
			Message: err.Error(),
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

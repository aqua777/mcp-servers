package git

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// gitStatus returns the working tree status formatted like `git status`.
func gitStatus(repo *gogit.Repository) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}
	return status.String(), nil
}

// gitDiffUnstaged shows changes in the working directory not yet staged.
func gitDiffUnstaged(repo *gogit.Repository, contextLines int) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("getting HEAD tree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	// Use status to identify worktree-modified files, then produce a simple diff
	var sb strings.Builder
	for path, s := range status {
		if s.Worktree != gogit.Unmodified && s.Worktree != gogit.Untracked {
			file, err := headTree.File(path)
			if err != nil {
				// New file in index — show as added in worktree
				continue
			}
			oldContent, err := file.Contents()
			if err != nil {
				continue
			}
			worktreeFile, err := wt.Filesystem.Open(path)
			if err != nil {
				continue
			}
			var newBuf bytes.Buffer
			_, _ = newBuf.ReadFrom(worktreeFile)
			worktreeFile.Close()
			newContent := newBuf.String()
			sb.WriteString(unifiedDiff("a/"+path, "b/"+path, oldContent, newContent, contextLines))
		}
	}
	return sb.String(), nil
}

// gitDiffStaged shows changes that are staged for commit (index vs HEAD).
func gitDiffStaged(repo *gogit.Repository, contextLines int) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("getting HEAD tree: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("getting status: %w", err)
	}

	var sb strings.Builder
	for path, s := range status {
		if s.Staging == gogit.Unmodified || s.Staging == gogit.Untracked {
			continue
		}
		var oldContent string
		if headFile, err := headTree.File(path); err == nil {
			oldContent, _ = headFile.Contents()
		}
		// Read staged content from the index via the object store
		idx, err := repo.Storer.Index()
		if err != nil {
			continue
		}
		var newContent string
		for _, entry := range idx.Entries {
			if entry.Name == path {
				obj, err := repo.BlobObject(entry.Hash)
				if err != nil {
					break
				}
				r, err := obj.Reader()
				if err != nil {
					break
				}
				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				r.Close()
				newContent = buf.String()
				break
			}
		}
		sb.WriteString(unifiedDiff("a/"+path, "b/"+path, oldContent, newContent, contextLines))
	}
	return sb.String(), nil
}

// gitDiff shows differences between the current HEAD and a target ref.
func gitDiff(repo *gogit.Repository, target string, contextLines int) (string, error) {
	if err := validateRefName(target); err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit: %w", err)
	}

	targetHash, err := repo.ResolveRevision(plumbing.Revision(target))
	if err != nil {
		return "", fmt.Errorf("resolving target %q: %w", target, err)
	}
	targetCommit, err := repo.CommitObject(*targetHash)
	if err != nil {
		return "", fmt.Errorf("getting target commit: %w", err)
	}

	patch, err := targetCommit.Patch(headCommit)
	if err != nil {
		return "", fmt.Errorf("computing diff: %w", err)
	}

	return encodePatch(patch, contextLines), nil
}

// gitCommit records staged changes to the repository.
func gitCommit(repo *gogit.Repository, message string) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "MCP Git Server",
			Email: "mcp@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}
	return fmt.Sprintf("Changes committed successfully with hash %s", hash.String()), nil
}

// gitAdd stages files for commit.
func gitAdd(repo *gogit.Repository, files []string) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	if len(files) == 1 && files[0] == "." {
		if err := wt.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
			return "", fmt.Errorf("staging all files: %w", err)
		}
	} else {
		for _, f := range files {
			if _, err := wt.Add(f); err != nil {
				return "", fmt.Errorf("staging %q: %w", f, err)
			}
		}
	}
	return "Files staged successfully", nil
}

// gitReset unstages all staged changes (mixed reset to HEAD).
func gitReset(repo *gogit.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	if err := wt.Reset(&gogit.ResetOptions{
		Commit: head.Hash(),
		Mode:   gogit.MixedReset,
	}); err != nil {
		return "", fmt.Errorf("resetting: %w", err)
	}
	return "All staged changes reset", nil
}

// gitLog returns commit history entries.
func gitLog(repo *gogit.Repository, maxCount int, startTimestamp, endTimestamp string) ([]string, error) {
	opts := &gogit.LogOptions{
		Order: gogit.LogOrderCommitterTime,
	}

	if startTimestamp != "" {
		t, err := parseTimestamp(startTimestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid start_timestamp: %w", err)
		}
		opts.Since = &t
	}
	if endTimestamp != "" {
		t, err := parseTimestamp(endTimestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid end_timestamp: %w", err)
		}
		opts.Until = &t
	}

	iter, err := repo.Log(opts)
	if err != nil {
		return nil, fmt.Errorf("getting log: %w", err)
	}
	defer iter.Close()

	var entries []string
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if maxCount > 0 && count >= maxCount {
			return fmt.Errorf("stop") // sentinel to stop iteration
		}
		entries = append(entries, fmt.Sprintf(
			"Commit: %q\nAuthor: %q\nDate: %s\nMessage: %q\n",
			c.Hash.String(),
			c.Author.Name,
			c.Author.When.Format(time.RFC3339),
			strings.TrimRight(c.Message, "\n"),
		))
		count++
		return nil
	})
	// Ignore our sentinel stop error
	if err != nil && err.Error() != "stop" {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}
	return entries, nil
}

// gitCreateBranch creates a new branch from baseBranch (or HEAD).
func gitCreateBranch(repo *gogit.Repository, branchName, baseBranch string) (string, error) {
	var baseRef *plumbing.Reference
	if baseBranch != "" {
		ref, err := repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
		if err != nil {
			return "", fmt.Errorf("resolving base branch %q: %w", baseBranch, err)
		}
		baseRef = ref
	} else {
		ref, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("getting HEAD: %w", err)
		}
		baseRef = ref
	}

	newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), baseRef.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", fmt.Errorf("creating branch %q: %w", branchName, err)
	}

	baseName := baseBranch
	if baseName == "" {
		baseName = baseRef.Name().Short()
	}
	return fmt.Sprintf("Created branch %q from %q", branchName, baseName), nil
}

// gitCheckout switches branches.
func gitCheckout(repo *gogit.Repository, branchName string) (string, error) {
	if err := validateRefName(branchName); err != nil {
		return "", err
	}

	// Verify the branch exists
	_, err := repo.ResolveRevision(plumbing.Revision(branchName))
	if err != nil {
		return "", fmt.Errorf("branch %q not found: %w", branchName, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	if err := wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	}); err != nil {
		return "", fmt.Errorf("checking out %q: %w", branchName, err)
	}
	return fmt.Sprintf("Switched to branch %q", branchName), nil
}

// gitShow shows the contents of a commit (metadata + diff).
func gitShow(repo *gogit.Repository, revision string) (string, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return "", fmt.Errorf("resolving revision %q: %w", revision, err)
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return "", fmt.Errorf("getting commit: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Commit: %q\nAuthor: %q\nDate: %s\nMessage: %q\n",
		commit.Hash.String(),
		commit.Author.Name,
		commit.Author.When.Format(time.RFC3339),
		strings.TrimRight(commit.Message, "\n"),
	)

	var patch *object.Patch
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return "", fmt.Errorf("getting parent commit: %w", err)
		}
		patch, err = parent.Patch(commit)
		if err != nil {
			return "", fmt.Errorf("computing patch: %w", err)
		}
	} else {
		// Initial commit: diff against empty tree
		emptyTree := &object.Tree{}
		commitTree, err := commit.Tree()
		if err != nil {
			return "", fmt.Errorf("getting commit tree: %w", err)
		}
		patch, err = emptyTree.Patch(commitTree)
		if err != nil {
			return "", fmt.Errorf("computing initial commit patch: %w", err)
		}
	}

	sb.WriteString(encodePatch(patch, DefaultContextLines))
	return sb.String(), nil
}

// gitBranch lists branches filtered by type and optional contains/not-contains SHA.
func gitBranch(repo *gogit.Repository, branchType, contains, notContains string) (string, error) {
	refs, err := repo.References()
	if err != nil {
		return "", fmt.Errorf("listing references: %w", err)
	}

	var containsHash, notContainsHash plumbing.Hash
	if contains != "" {
		containsHash = plumbing.NewHash(contains)
	}
	if notContains != "" {
		notContainsHash = plumbing.NewHash(notContains)
	}

	var names []string
	var currentBranch string
	if head, err := repo.Head(); err == nil {
		currentBranch = head.Name().Short()
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		switch branchType {
		case "local":
			if !name.IsBranch() {
				return nil
			}
		case "remote":
			if !name.IsRemote() {
				return nil
			}
		case "all":
			if !name.IsBranch() && !name.IsRemote() {
				return nil
			}
		default:
			return fmt.Errorf("invalid branch_type %q: must be 'local', 'remote', or 'all'", branchType)
		}

		// contains filter
		if contains != "" {
			if !refContainsCommit(repo, ref.Hash(), containsHash) {
				return nil
			}
		}
		// not_contains filter
		if notContains != "" {
			if refContainsCommit(repo, ref.Hash(), notContainsHash) {
				return nil
			}
		}

		shortName := name.Short()
		prefix := "  "
		if shortName == currentBranch && name.IsBranch() {
			prefix = "* "
		}
		names = append(names, prefix+shortName)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(names)
	return strings.Join(names, "\n"), nil
}

// refContainsCommit checks whether the commit reachable from refHash contains targetHash in its ancestry.
func refContainsCommit(repo *gogit.Repository, refHash, targetHash plumbing.Hash) bool {
	iter, err := repo.Log(&gogit.LogOptions{From: refHash, Order: gogit.LogOrderCommitterTime})
	if err != nil {
		return false
	}
	defer iter.Close()
	found := false
	_ = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == targetHash {
			found = true
			return fmt.Errorf("stop")
		}
		return nil
	})
	return found
}

// encodePatch encodes a go-git Patch using the unified encoder with the given context lines.
func encodePatch(patch *object.Patch, contextLines int) string {
	var buf bytes.Buffer
	encoder := diff.NewUnifiedEncoder(&buf, contextLines)
	if err := encoder.Encode(patch); err != nil {
		return patch.String()
	}
	return buf.String()
}

// unifiedDiff produces a minimal unified diff between oldContent and newContent.
func unifiedDiff(oldName, newName, oldContent, newContent string, contextLines int) string {
	if oldContent == newContent {
		return ""
	}
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", oldName, newName)

	// Simple line-by-line diff with context
	type change struct {
		kind rune // ' ', '+', '-'
		line string
	}
	var changes []change

	// Longest-common-subsequence not implemented here; use a simple diff approach
	// by producing a single hunk if files differ
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	// Build a simple Myers-like output: remove all old, add all new per hunk
	// For a production implementation this is simplified; see parity.md for known differences
	fmt.Fprintf(&sb, "@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines))
	for _, l := range oldLines {
		changes = append(changes, change{'-', l})
	}
	for _, l := range newLines {
		changes = append(changes, change{'+', l})
	}
	_ = contextLines // not applicable with this simple approach

	for _, c := range changes {
		fmt.Fprintf(&sb, "%c%s\n", c.kind, c.line)
	}
	return sb.String()
}

// parseTimestamp parses a timestamp string in various formats.
// Supports ISO 8601, date-only (YYYY-MM-DD), and relative formats via dateparse library.
func parseTimestamp(s string) (time.Time, error) {
	// Try standard Go formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"Jan 2 2006",
		"January 2 2006",
		"Jan 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	// Try relative formats like "2 weeks ago", "yesterday"
	t, err := parseRelativeTime(s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", s)
}

// parseRelativeTime handles simple relative time strings.
func parseRelativeTime(s string) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	now := time.Now()

	switch s {
	case "today":
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, now.Location()), nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}

	// "N unit(s) ago"
	var n int
	var unit string
	if _, err := fmt.Sscanf(s, "%d %s", &n, &unit); err != nil {
		return time.Time{}, fmt.Errorf("unrecognized relative time: %q", s)
	}
	unit = strings.TrimSuffix(unit, " ago")
	unit = strings.TrimSuffix(unit, "s")

	switch unit {
	case "minute":
		return now.Add(-time.Duration(n) * time.Minute), nil
	case "hour":
		return now.Add(-time.Duration(n) * time.Hour), nil
	case "day":
		return now.AddDate(0, 0, -n), nil
	case "week":
		return now.AddDate(0, 0, -n*7), nil
	case "month":
		return now.AddDate(0, -n, 0), nil
	case "year":
		return now.AddDate(-n, 0, 0), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time unit: %q", unit)
}

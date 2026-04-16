package git

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// computeAheadBehind calculates how many commits local is ahead/behind remote.
func computeAheadBehind(repo *gogit.Repository, localHash, remoteHash plumbing.Hash) (ahead, behind int, err error) {
	if localHash == remoteHash {
		return 0, 0, nil
	}

	localCommit, err := repo.CommitObject(localHash)
	if err != nil {
		return 0, 0, fmt.Errorf("resolving local commit: %w", err)
	}
	remoteCommit, err := repo.CommitObject(remoteHash)
	if err != nil {
		return 0, 0, fmt.Errorf("resolving remote commit: %w", err)
	}

	localAncestors := make(map[plumbing.Hash]bool)
	iter := object.NewCommitIterCTime(localCommit, nil, nil)
	_ = iter.ForEach(func(c *object.Commit) error {
		localAncestors[c.Hash] = true
		return nil
	})

	remoteAncestors := make(map[plumbing.Hash]bool)
	iter = object.NewCommitIterCTime(remoteCommit, nil, nil)
	_ = iter.ForEach(func(c *object.Commit) error {
		remoteAncestors[c.Hash] = true
		return nil
	})

	for hash := range localAncestors {
		if !remoteAncestors[hash] {
			ahead++
		}
	}

	for hash := range remoteAncestors {
		if !localAncestors[hash] {
			behind++
		}
	}

	return ahead, behind, nil
}

// buildRefMap builds a map from commit hash to decorated ref names (like git log --decorate).
func buildRefMap(repo *gogit.Repository) map[plumbing.Hash][]string {
	refMap := map[plumbing.Hash][]string{}
	refs, err := repo.References()
	if err != nil {
		return refMap
	}
	defer refs.Close()

	head, _ := repo.Head()
	var headHash plumbing.Hash
	var headBranch string
	if head != nil {
		headHash = head.Hash()
		headBranch = head.Name().Short()
	}

	_ = refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		if !name.IsBranch() && !name.IsRemote() && !name.IsTag() {
			return nil
		}
		short := name.Short()
		if name.IsTag() {
			short = "tag: " + short
		}
		refMap[ref.Hash()] = append(refMap[ref.Hash()], short)
		return nil
	})

	if head != nil {
		existing := refMap[headHash]
		var filtered []string
		for _, r := range existing {
			if r != headBranch {
				filtered = append(filtered, r)
			}
		}
		refMap[headHash] = append([]string{"HEAD -> " + headBranch}, filtered...)
	}

	return refMap
}

// commitToInfo converts a go-git Commit to a CommitInfo struct.
func commitToInfo(c *object.Commit, refMap map[plumbing.Hash][]string) CommitInfo {
	var parents []string
	for _, p := range c.ParentHashes {
		parents = append(parents, p.String())
	}
	refs := refMap[c.Hash]
	if refs == nil {
		refs = []string{}
	}
	if parents == nil {
		parents = []string{}
	}
	return CommitInfo{
		SHA: c.Hash.String(),
		Author: AuthorInfo{
			Name:  c.Author.Name,
			Email: c.Author.Email,
		},
		Date:    c.Author.When.Format(time.RFC3339),
		Message: strings.TrimRight(c.Message, "\n"),
		Refs:    refs,
		Parents: parents,
	}
}

// statusCodeToString maps go-git StatusCode to a human-readable status string.
func statusCodeToString(code gogit.StatusCode) string {
	switch code {
	case gogit.Added:
		return "added"
	case gogit.Modified:
		return "modified"
	case gogit.Deleted:
		return "deleted"
	case gogit.Renamed:
		return "renamed"
	case gogit.Copied:
		return "copied"
	default:
		return "modified"
	}
}

// sortFileChanges sorts a slice of FileChange by path.
func sortFileChanges(changes []FileChange) {
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})
}

// patchToDiffResult converts a go-git Patch to a structured DiffResult.
func patchToDiffResult(patch *object.Patch, base, target string, contextLines, maxFiles int) *DiffResult {
	result := &DiffResult{
		Status:  "no_changes",
		Base:    base,
		Target:  target,
		Files:   []DiffFile{},
		Summary: DiffSummary{},
		RawText: encodePatch(patch, contextLines),
	}

	fps := patch.FilePatches()
	if len(fps) == 0 {
		return result
	}

	result.Status = "has_changes"
	for _, fp := range fps {
		from, to := fp.Files()
		df := DiffFile{
			Changes: []DiffChange{},
		}

		if from == nil && to != nil {
			df.Path = to.Path()
			df.Status = "added"
		} else if from != nil && to == nil {
			df.Path = from.Path()
			df.Status = "deleted"
		} else if from != nil && to != nil {
			df.Path = to.Path()
			if from.Path() != to.Path() {
				df.OldPath = from.Path()
				df.Status = "renamed"
			} else {
				df.Status = "modified"
			}
		}

		df.Binary = fp.IsBinary()

		if !fp.IsBinary() {
			changes, adds, dels := parseDiffChunks(fp.Chunks(), contextLines)
			df.Changes = changes
			df.Additions = adds
			df.Deletions = dels
		}

		if maxFiles > 0 && len(result.Files) >= maxFiles {
			result.Truncated = true
			break
		}
		result.Summary.TotalAdditions += df.Additions
		result.Summary.TotalDeletions += df.Deletions
		result.Files = append(result.Files, df)
	}
	result.Summary.TotalFiles = len(result.Files)

	return result
}

// parseDiffChunks processes diff chunks into structured DiffChange entries.
func parseDiffChunks(chunks []diff.Chunk, contextLines int) ([]DiffChange, int, int) {
	var changes []DiffChange
	var additions, deletions int
	oldLine := 1
	newLine := 1

	var contextBuffer []string

	for _, chunk := range chunks {
		lines := splitChunkLines(chunk.Content())

		switch chunk.Type() {
		case diff.Equal:
			for _, line := range lines {
				contextBuffer = append(contextBuffer, line)
				if len(contextBuffer) > contextLines {
					contextBuffer = contextBuffer[1:]
				}
				changes = append(changes, DiffChange{
					Type:       "context",
					OldLine:    oldLine,
					NewLine:    newLine,
					OldContent: line,
				})
				oldLine++
				newLine++
			}
		case diff.Delete:
			var ctxBefore []string
			if len(contextBuffer) > 0 {
				ctxBefore = make([]string, len(contextBuffer))
				copy(ctxBefore, contextBuffer)
			}
			for _, line := range lines {
				ch := DiffChange{
					Type:          "deletion",
					OldLine:       oldLine,
					OldContent:    line,
					ContextBefore: ctxBefore,
				}
				changes = append(changes, ch)
				deletions++
				oldLine++
				ctxBefore = nil
			}
			contextBuffer = nil
		case diff.Add:
			var ctxBefore []string
			if len(contextBuffer) > 0 {
				ctxBefore = make([]string, len(contextBuffer))
				copy(ctxBefore, contextBuffer)
			}
			for _, line := range lines {
				ch := DiffChange{
					Type:          "addition",
					NewLine:       newLine,
					NewContent:    line,
					ContextBefore: ctxBefore,
				}
				changes = append(changes, ch)
				additions++
				newLine++
				ctxBefore = nil
			}
			contextBuffer = nil
		}
	}

	// Backward pass: populate context_after for additions and deletions
	for i := 0; i < len(changes); i++ {
		if changes[i].Type == "addition" || changes[i].Type == "deletion" {
			var ctxAfter []string
			for j := i + 1; j < len(changes) && len(ctxAfter) < contextLines; j++ {
				if changes[j].Type == "context" {
					ctxAfter = append(ctxAfter, changes[j].OldContent)
				} else {
					break
				}
			}
			if len(ctxAfter) > 0 {
				changes[i].ContextAfter = ctxAfter
			}
		}
	}

	return changes, additions, deletions
}

// splitChunkLines splits chunk content into lines, removing a trailing empty line from the split.
func splitChunkLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// manualDiffToFile produces a DiffFile from old/new content using the simple all-delete/all-add approach.
func manualDiffToFile(path, oldContent, newContent, status string, contextLines int) *DiffFile {
	if oldContent == newContent {
		return nil
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	df := &DiffFile{
		Path:    path,
		Status:  status,
		Changes: []DiffChange{},
	}

	df.Changes = append(df.Changes, DiffChange{
		Type:       "hunk_header",
		NewContent: fmt.Sprintf("@@ -1,%d +1,%d @@", len(oldLines), len(newLines)),
	})

	for i, l := range oldLines {
		df.Changes = append(df.Changes, DiffChange{
			Type:       "deletion",
			OldLine:    i + 1,
			OldContent: l,
		})
		df.Deletions++
	}
	for i, l := range newLines {
		df.Changes = append(df.Changes, DiffChange{
			Type:       "addition",
			NewLine:    i + 1,
			NewContent: l,
		})
		df.Additions++
	}

	return df
}

// gitStatus returns the working tree status as a structured StatusResult.
func gitStatus(repo *gogit.Repository) (*StatusResult, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}

	result := &StatusResult{
		Changes: StatusChanges{
			Staged:    []FileChange{},
			Unstaged:  []FileChange{},
			Untracked: []FileChange{},
			Conflicts: []FileChange{},
		},
	}

	head, err := repo.Head()
	if err == nil {
		result.Repository.Branch = head.Name().Short()
		result.Repository.HeadSHA = head.Hash().String()

		cfg, cfgErr := repo.Config()
		if cfgErr == nil {
			if branchCfg, ok := cfg.Branches[result.Repository.Branch]; ok && branchCfg.Remote != "" {
				remoteBranch := branchCfg.Remote + "/" + branchCfg.Merge.Short()
				remoteRef := plumbing.NewRemoteReferenceName(branchCfg.Remote, branchCfg.Merge.Short())
				remoteRefObj, refErr := repo.Reference(remoteRef, true)
				if refErr == nil {
					ahead, behind, abErr := computeAheadBehind(repo, head.Hash(), remoteRefObj.Hash())
					if abErr == nil {
						status := "up_to_date"
						if ahead > 0 && behind > 0 {
							status = "diverged"
						} else if ahead > 0 {
							status = "ahead"
						} else if behind > 0 {
							status = "behind"
						}
						result.Repository.Remote = &RemoteInfo{
							Name:     branchCfg.Remote,
							Branch:   remoteBranch,
							Status:   status,
							AheadBy:  ahead,
							BehindBy: behind,
						}
					}
				}
			}
		}
	}

	for path, s := range status {
		if s.Staging == gogit.UpdatedButUnmerged || s.Worktree == gogit.UpdatedButUnmerged {
			result.Changes.Conflicts = append(result.Changes.Conflicts, FileChange{
				Path:   path,
				Status: "conflict",
			})
			continue
		}

		if s.Staging != gogit.Unmodified && s.Staging != gogit.Untracked {
			result.Changes.Staged = append(result.Changes.Staged, FileChange{
				Path:   path,
				Status: statusCodeToString(s.Staging),
			})
		}

		if s.Worktree == gogit.Untracked {
			fileType := "file"
			if info, statErr := os.Stat(filepath.Join(wt.Filesystem.Root(), path)); statErr == nil && info.IsDir() {
				fileType = "directory"
			}
			result.Changes.Untracked = append(result.Changes.Untracked, FileChange{
				Path:   path,
				Status: "untracked",
				Type:   fileType,
			})
		} else if s.Worktree != gogit.Unmodified {
			result.Changes.Unstaged = append(result.Changes.Unstaged, FileChange{
				Path:   path,
				Status: statusCodeToString(s.Worktree),
			})
		}
	}

	sortFileChanges(result.Changes.Staged)
	sortFileChanges(result.Changes.Unstaged)
	sortFileChanges(result.Changes.Untracked)
	sortFileChanges(result.Changes.Conflicts)

	result.Summary = StatusSummary{
		StagedCount:     len(result.Changes.Staged),
		UnstagedCount:   len(result.Changes.Unstaged),
		UntrackedCount:  len(result.Changes.Untracked),
		ConflictedCount: len(result.Changes.Conflicts),
	}
	result.Summary.TotalFiles = result.Summary.StagedCount + result.Summary.UnstagedCount +
		result.Summary.UntrackedCount + result.Summary.ConflictedCount

	return result, nil
}

// gitDiffUnstaged shows changes in the working directory not yet staged.
func gitDiffUnstaged(repo *gogit.Repository, contextLines, maxFiles int) (*DiffResult, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("getting HEAD commit: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD tree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}

	result := &DiffResult{
		Status:  "no_changes",
		Base:    "HEAD",
		Target:  "working tree",
		Files:   []DiffFile{},
		Summary: DiffSummary{},
	}

	var rawSB strings.Builder
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

			rawSB.WriteString(unifiedDiff("a/"+path, "b/"+path, oldContent, newContent, contextLines))

			df := manualDiffToFile(path, oldContent, newContent, statusCodeToString(s.Worktree), contextLines)
			if df != nil {
				if maxFiles > 0 && len(result.Files) >= maxFiles {
					result.Truncated = true
					break
				}
				result.Files = append(result.Files, *df)
				result.Summary.TotalAdditions += df.Additions
				result.Summary.TotalDeletions += df.Deletions
			}
		}
	}

	if len(result.Files) > 0 {
		result.Status = "has_changes"
		result.Summary.TotalFiles = len(result.Files)
		result.RawText = rawSB.String()
		sort.Slice(result.Files, func(i, j int) bool {
			return result.Files[i].Path < result.Files[j].Path
		})
	}

	return result, nil
}

// gitDiffStaged shows changes that are staged for commit (index vs HEAD).
func gitDiffStaged(repo *gogit.Repository, contextLines, maxFiles int) (*DiffResult, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("getting HEAD commit: %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD tree: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}

	result := &DiffResult{
		Status:  "no_changes",
		Base:    "HEAD",
		Target:  "index",
		Files:   []DiffFile{},
		Summary: DiffSummary{},
	}

	var rawSB strings.Builder
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

		rawSB.WriteString(unifiedDiff("a/"+path, "b/"+path, oldContent, newContent, contextLines))

		df := manualDiffToFile(path, oldContent, newContent, statusCodeToString(s.Staging), contextLines)
		if df != nil {
			if maxFiles > 0 && len(result.Files) >= maxFiles {
				result.Truncated = true
				break
			}
			result.Files = append(result.Files, *df)
			result.Summary.TotalAdditions += df.Additions
			result.Summary.TotalDeletions += df.Deletions
		}
	}

	if len(result.Files) > 0 {
		result.Status = "has_changes"
		result.Summary.TotalFiles = len(result.Files)
		result.RawText = rawSB.String()
		sort.Slice(result.Files, func(i, j int) bool {
			return result.Files[i].Path < result.Files[j].Path
		})
	}

	return result, nil
}

// gitDiff shows differences between the current HEAD and a target ref.
func gitDiff(repo *gogit.Repository, target string, contextLines, maxFiles int) (*DiffResult, error) {
	if err := validateRefName(target); err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD: %w", err)
	}
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("getting HEAD commit: %w", err)
	}

	targetHash, err := repo.ResolveRevision(plumbing.Revision(target))
	if err != nil {
		return nil, fmt.Errorf("resolving target %q: %w", target, err)
	}
	targetCommit, err := repo.CommitObject(*targetHash)
	if err != nil {
		return nil, fmt.Errorf("getting target commit: %w", err)
	}

	patch, err := targetCommit.Patch(headCommit)
	if err != nil {
		return nil, fmt.Errorf("computing diff: %w", err)
	}

	return patchToDiffResult(patch, target, "HEAD", contextLines, maxFiles), nil
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

// gitLog returns commit history as a structured LogResult.
func gitLog(repo *gogit.Repository, maxCount int, startTimestamp, endTimestamp string) (*LogResult, error) {
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

	refMap := buildRefMap(repo)

	iter, err := repo.Log(opts)
	if err != nil {
		return nil, fmt.Errorf("getting log: %w", err)
	}
	defer iter.Close()

	result := &LogResult{Commits: []CommitInfo{}}
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if maxCount > 0 && count >= maxCount {
			return fmt.Errorf("stop") // sentinel to stop iteration
		}
		result.Commits = append(result.Commits, commitToInfo(c, refMap))
		count++
		return nil
	})
	// Ignore our sentinel stop error
	if err != nil && err.Error() != "stop" {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}
	return result, nil
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

// gitShow shows the contents of a commit (metadata + diff) as a structured ShowResult.
func gitShow(repo *gogit.Repository, revision string) (*ShowResult, error) {
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return nil, fmt.Errorf("resolving revision %q: %w", revision, err)
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, fmt.Errorf("getting commit: %w", err)
	}

	refMap := buildRefMap(repo)

	var patch *object.Patch
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("getting parent commit: %w", err)
		}
		patch, err = parent.Patch(commit)
		if err != nil {
			return nil, fmt.Errorf("computing patch: %w", err)
		}
	} else {
		// Initial commit: diff against empty tree
		emptyTree := &object.Tree{}
		commitTree, err := commit.Tree()
		if err != nil {
			return nil, fmt.Errorf("getting commit tree: %w", err)
		}
		patch, err = emptyTree.Patch(commitTree)
		if err != nil {
			return nil, fmt.Errorf("computing initial commit patch: %w", err)
		}
	}

	result := &ShowResult{
		Commit: commitToInfo(commit, refMap),
		Diff:   *patchToDiffResult(patch, "", "", DefaultContextLines, 0),
	}
	return result, nil
}

// gitBranch lists branches as a structured BranchResult.
func gitBranch(repo *gogit.Repository, branchType, contains, notContains string) (*BranchResult, error) {
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}

	var containsHash, notContainsHash plumbing.Hash
	if contains != "" {
		containsHash = plumbing.NewHash(contains)
	}
	if notContains != "" {
		notContainsHash = plumbing.NewHash(notContains)
	}

	result := &BranchResult{
		Branches: []BranchInfo{},
	}

	var currentBranch string
	if head, err := repo.Head(); err == nil {
		if head.Name().IsBranch() {
			currentBranch = head.Name().Short()
			result.CurrentBranch = currentBranch
		} else {
			result.IsDetached = true
			result.CurrentBranch = head.Hash().String()[:7]
		}
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
		isCurrent := shortName == currentBranch && name.IsBranch()

		bi := BranchInfo{
			Name:          shortName,
			IsCurrent:     isCurrent,
			LastCommitSHA: ref.Hash().String(),
		}

		if commit, err := repo.CommitObject(ref.Hash()); err == nil {
			bi.LastCommitDate = commit.Author.When.Format(time.RFC3339)
		}

		if name.IsBranch() {
			if cfg, err := repo.Config(); err == nil {
				if branchCfg, ok := cfg.Branches[shortName]; ok && branchCfg.Remote != "" {
					bi.Tracking = branchCfg.Remote + "/" + branchCfg.Merge.Short()
				}
			}
		}

		result.Branches = append(result.Branches, bi)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result.Branches, func(i, j int) bool {
		return result.Branches[i].Name < result.Branches[j].Name
	})

	return result, nil
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

package filesystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dlclark/regexp2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	engineRE2   = "re2"
	enginePCRE2 = "pcre2"

	defaultMaxMatches = 1000

	binarySniffLen = 512
)

func (s *FilesystemServer) registerGrepTools() {
	s.server.AddTool(&mcp.Tool{
		Name: ToolGrep,
		Description: `Search file contents for a regex or literal pattern, recursively across a directory or within a single file.
Equivalent to ripgrep (rg). Returns matching lines with file path and line number.
Supports context lines, glob include/exclude filters, case controls, and two regex engines (RE2 and PCRE2).`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory or file to search (must be within allowed directories)",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Search pattern (regex by default; use fixedStrings for literal match)",
				},
				"fixedStrings": map[string]any{
					"type":        "boolean",
					"description": "Treat pattern as a literal string, not a regex (default: false)",
					"default":     false,
				},
				"ignoreCase": map[string]any{
					"type":        "boolean",
					"description": "Case-insensitive search (default: false)",
					"default":     false,
				},
				"smartCase": map[string]any{
					"type":        "boolean",
					"description": "Case-insensitive unless pattern contains uppercase letters (default: false)",
					"default":     false,
				},
				"engine": map[string]any{
					"type":        "string",
					"enum":        []string{engineRE2, enginePCRE2},
					"description": "Regex engine: 're2' (default, Go stdlib) or 'pcre2' (supports lookahead/backreferences)",
				},
				"includePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Glob patterns: only files matching at least one pattern are searched (e.g. ['*.go', '*.ts'])",
				},
				"excludePatterns": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Glob patterns: files matching any pattern are skipped (e.g. ['vendor/**', '*.min.js'])",
				},
				"contextBefore": map[string]any{
					"type":        "integer",
					"description": "Number of lines of context to show before each match",
					"minimum":     0,
				},
				"contextAfter": map[string]any{
					"type":        "integer",
					"description": "Number of lines of context to show after each match",
					"minimum":     0,
				},
				"contextLines": map[string]any{
					"type":        "integer",
					"description": "Symmetric context lines before and after each match (overridden by contextBefore/contextAfter)",
					"minimum":     0,
				},
				"maxMatches": map[string]any{
					"type":        "integer",
					"description": fmt.Sprintf("Maximum number of matches to return (default: %d, 0 = unlimited)", defaultMaxMatches),
					"minimum":     0,
				},
				"format": map[string]any{
					"type":        "string",
					"enum":        []string{"text", "json"},
					"description": "Output format (default: server setting)",
				},
			},
			"required": []string{"path", "pattern"},
		},
	}, s.handleGrep)
}

// grepArgs holds the parsed arguments for the grep tool.
type grepArgs struct {
	Path            string   `json:"path"`
	Pattern         string   `json:"pattern"`
	FixedStrings    bool     `json:"fixedStrings"`
	IgnoreCase      bool     `json:"ignoreCase"`
	SmartCase       bool     `json:"smartCase"`
	Engine          string   `json:"engine"`
	IncludePatterns []string `json:"includePatterns"`
	ExcludePatterns []string `json:"excludePatterns"`
	ContextBefore   *int     `json:"contextBefore"`
	ContextAfter    *int     `json:"contextAfter"`
	ContextLines    *int     `json:"contextLines"`
	MaxMatches      *int     `json:"maxMatches"`
	Format          string   `json:"format"`
}

// matcher is an abstraction over RE2 and PCRE2 engines.
type matcher interface {
	matchString(s string) bool
}

type re2Matcher struct{ re *regexp.Regexp }

func (m *re2Matcher) matchString(s string) bool { return m.re.MatchString(s) }

type pcre2Matcher struct{ re *regexp2.Regexp }

func (m *pcre2Matcher) matchString(s string) bool {
	ok, _ := m.re.MatchString(s)
	return ok
}

func (s *FilesystemServer) handleGrep(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args grepArgs
	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return errorResult(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	format := s.resolveFormat(args.Format)

	validPath, err := s.validatePath(args.Path)
	if err != nil {
		return errorResult(err.Error()), nil
	}

	// Resolve engine
	engine := args.Engine
	if engine == "" {
		engine = engineRE2
	}

	// Resolve case sensitivity
	ignoreCase := args.IgnoreCase
	if !ignoreCase && args.SmartCase {
		ignoreCase = !patternHasUppercase(args.Pattern)
	}

	// Build effective pattern
	effectivePattern := args.Pattern
	if args.FixedStrings {
		effectivePattern = regexp.QuoteMeta(effectivePattern)
	}
	if ignoreCase && engine == engineRE2 {
		effectivePattern = "(?i)" + effectivePattern
	}

	// Compile matcher
	var m matcher
	switch engine {
	case enginePCRE2:
		opts := regexp2.None
		if ignoreCase {
			opts |= regexp2.IgnoreCase
		}
		re, err := regexp2.Compile(effectivePattern, opts)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid PCRE2 pattern: %v", err)), nil
		}
		m = &pcre2Matcher{re: re}
	default:
		re, err := regexp.Compile(effectivePattern)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid regex pattern: %v", err)), nil
		}
		m = &re2Matcher{re: re}
	}

	// Resolve context lines
	ctxBefore := 0
	ctxAfter := 0
	if args.ContextLines != nil {
		ctxBefore = *args.ContextLines
		ctxAfter = *args.ContextLines
	}
	if args.ContextBefore != nil {
		ctxBefore = *args.ContextBefore
	}
	if args.ContextAfter != nil {
		ctxAfter = *args.ContextAfter
	}

	// Resolve max matches
	maxMatches := defaultMaxMatches
	if args.MaxMatches != nil {
		maxMatches = *args.MaxMatches
	}

	// Determine whether path is a file or directory
	info, err := os.Stat(validPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Cannot access path: %v", err)), nil
	}

	var matches []GrepMatch
	filesSearched := 0
	filesMatched := 0
	truncated := false

	collectFromFile := func(filePath string) bool {
		fileMatches, err := searchFile(filePath, m, ctxBefore, ctxAfter)
		if err != nil {
			return true // skip unreadable / binary files silently
		}
		filesSearched++
		if len(fileMatches) == 0 {
			return true
		}
		filesMatched++
		for _, fm := range fileMatches {
			if maxMatches > 0 && len(matches) >= maxMatches {
				truncated = true
				return false
			}
			matches = append(matches, fm)
		}
		return true
	}

	if !info.IsDir() {
		collectFromFile(validPath)
	} else {
		fileSys := os.DirFS(validPath)
		_ = fs.WalkDir(fileSys, ".", func(relPath string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			absPath := filepath.Join(validPath, filepath.FromSlash(relPath))

			// Apply exclude patterns
			for _, excl := range args.ExcludePatterns {
				matched, _ := doublestar.Match(excl, relPath)
				if matched {
					return nil
				}
			}

			// Apply include patterns (if any)
			if len(args.IncludePatterns) > 0 {
				included := false
				for _, incl := range args.IncludePatterns {
					matched, _ := doublestar.Match(incl, relPath)
					if matched {
						included = true
						break
					}
				}
				if !included {
					return nil
				}
			}

			if !collectFromFile(absPath) {
				return fs.SkipAll
			}
			return nil
		})
	}

	result := &GrepResult{
		Pattern: args.Pattern,
		Root:    validPath,
		Matches: matches,
		Summary: GrepSummary{
			TotalMatches:  len(matches),
			FilesMatched:  filesMatched,
			FilesSearched: filesSearched,
			Truncated:     truncated,
			Engine:        engine,
		},
	}

	var output string
	if format == FormatJSON {
		output = formatGrepJSON(result)
	} else {
		output = formatGrepText(result)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil
}

// searchFile reads a file and returns all matching lines with optional context.
// Returns an error (and no matches) if the file is binary or unreadable.
func searchFile(filePath string, m matcher, ctxBefore, ctxAfter int) ([]GrepMatch, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Binary detection: skip files with null bytes in first binarySniffLen bytes
	sniff := data
	if len(sniff) > binarySniffLen {
		sniff = sniff[:binarySniffLen]
	}
	if bytes.IndexByte(sniff, 0) >= 0 {
		return nil, fmt.Errorf("binary file")
	}

	lines := strings.Split(string(data), "\n")
	var matches []GrepMatch

	// Track which line indices have already been emitted as a match or context,
	// to avoid duplicating lines when match windows overlap.
	emitted := make(map[int]bool)

	for i, line := range lines {
		if !m.matchString(line) {
			continue
		}

		// Compute context windows
		beforeStart := i - ctxBefore
		if beforeStart < 0 {
			beforeStart = 0
		}
		afterEnd := i + ctxAfter
		if afterEnd >= len(lines) {
			afterEnd = len(lines) - 1
		}

		var before []ContextLine
		for j := beforeStart; j < i; j++ {
			before = append(before, ContextLine{LineNumber: j + 1, LineText: lines[j]})
		}

		var after []ContextLine
		for j := i + 1; j <= afterEnd; j++ {
			after = append(after, ContextLine{LineNumber: j + 1, LineText: lines[j]})
		}

		matches = append(matches, GrepMatch{
			Path:          filePath,
			LineNumber:    i + 1,
			LineText:      line,
			ContextBefore: before,
			ContextAfter:  after,
		})

		// Mark all lines in this match's window as emitted
		for j := beforeStart; j <= afterEnd; j++ {
			emitted[j] = true
		}
	}

	return matches, nil
}

// patternHasUppercase reports whether a pattern contains any uppercase Unicode letter.
func patternHasUppercase(pattern string) bool {
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

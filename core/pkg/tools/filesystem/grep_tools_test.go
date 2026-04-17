package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---- handler tests ----

func (s *FilesystemTestSuite) callGrep(args map[string]any) *mcp.CallToolResult {
	raw, _ := json.Marshal(args)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      ToolGrep,
			Arguments: raw,
		},
	}
	res, err := s.fsServer.handleGrep(s.ctx, req)
	s.Require().NoError(err)
	return res
}

func (s *FilesystemTestSuite) textOf(res *mcp.CallToolResult) string {
	s.Require().Len(res.Content, 1)
	return res.Content[0].(*mcp.TextContent).Text
}

// TestGrepBasicMatch verifies a simple literal string match inside a file.
func (s *FilesystemTestSuite) TestGrepBasicMatch() {
	f := s.createFile("basic.txt", "hello world\nfoo bar\nhello again\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "hello"})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "hello world")
	s.Contains(text, "hello again")
	s.NotContains(text, "foo bar")
}

// TestGrepNoMatches returns a human-readable "No matches found" in text mode.
func (s *FilesystemTestSuite) TestGrepNoMatches() {
	f := s.createFile("nomatch.txt", "line one\nline two\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "zzznomatch"})
	s.False(res.IsError)
	s.Contains(s.textOf(res), "No matches found")
}

// TestGrepFixedStrings treats regex metacharacters as literals.
func (s *FilesystemTestSuite) TestGrepFixedStrings() {
	f := s.createFile("fixed.txt", "price: $5.00\ncost: $3.00\nno dollar\n")

	// Without fixedStrings, "$5" in regex matches end-of-line (not literally)
	res := s.callGrep(map[string]any{"path": f, "pattern": "$5.00", "fixedStrings": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "price: $5.00")
	s.NotContains(text, "cost: $3.00")
}

// TestGrepIgnoreCase performs case-insensitive matching.
func (s *FilesystemTestSuite) TestGrepIgnoreCase() {
	f := s.createFile("case.txt", "Hello World\nHELLO WORLD\nhello world\nother line\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "hello", "ignoreCase": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "Hello World")
	s.Contains(text, "HELLO WORLD")
	s.Contains(text, "hello world")
	s.NotContains(text, "other line")
}

// TestGrepSmartCaseAllLower uses smart-case with a lowercase pattern → case-insensitive.
func (s *FilesystemTestSuite) TestGrepSmartCaseAllLower() {
	f := s.createFile("smart.txt", "Hello World\nhello world\nother\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "hello", "smartCase": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "Hello World")
	s.Contains(text, "hello world")
}

// TestGrepSmartCaseHasUpper uses smart-case with an uppercase pattern → case-sensitive.
func (s *FilesystemTestSuite) TestGrepSmartCaseHasUpper() {
	f := s.createFile("smartupper.txt", "Hello World\nhello world\nother\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "Hello", "smartCase": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "Hello World")
	s.NotContains(text, "hello world")
}

// TestGrepIgnoreCaseWinsOverSmartCase verifies ignoreCase overrides smartCase.
func (s *FilesystemTestSuite) TestGrepIgnoreCaseWinsOverSmartCase() {
	f := s.createFile("both.txt", "Hello World\nhello world\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "Hello", "ignoreCase": true, "smartCase": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "Hello World")
	s.Contains(text, "hello world")
}

// TestGrepRegexPattern verifies a real regex pattern works.
func (s *FilesystemTestSuite) TestGrepRegexPattern() {
	f := s.createFile("regex.txt", "func main() {\nvar x = 42\nfunc helper() {\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": `func \w+\(`})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "func main()")
	s.Contains(text, "func helper()")
	s.NotContains(text, "var x")
}

// TestGrepPCRE2Engine uses PCRE2 with a lookahead pattern.
func (s *FilesystemTestSuite) TestGrepPCRE2Engine() {
	f := s.createFile("pcre.txt", "foobar\nfoo123\nfoobaz\n")

	// Lookahead: foo followed by digits only
	res := s.callGrep(map[string]any{"path": f, "pattern": `foo(?=\d)`, "engine": "pcre2"})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "foo123")
	s.NotContains(text, "foobar")
	s.NotContains(text, "foobaz")
}

// TestGrepPCRE2IgnoreCase verifies PCRE2 engine respects ignoreCase.
func (s *FilesystemTestSuite) TestGrepPCRE2IgnoreCase() {
	f := s.createFile("pcre_ci.txt", "Hello\nhello\nHELLO\nother\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "hello", "engine": "pcre2", "ignoreCase": true})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "Hello")
	s.Contains(text, "hello")
	s.Contains(text, "HELLO")
	s.NotContains(text, "other")
}

// TestGrepInvalidRegex returns an error for an invalid RE2 pattern.
func (s *FilesystemTestSuite) TestGrepInvalidRegex() {
	f := s.createFile("invalid.txt", "some content\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "["})
	s.True(res.IsError)
	s.Contains(s.textOf(res), "Invalid regex pattern")
}

// TestGrepInvalidPCRE2Pattern returns an error for an invalid PCRE2 pattern.
func (s *FilesystemTestSuite) TestGrepInvalidPCRE2Pattern() {
	f := s.createFile("invalid2.txt", "some content\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "(?<=", "engine": "pcre2"})
	s.True(res.IsError)
	s.Contains(s.textOf(res), "Invalid PCRE2 pattern")
}

// TestGrepRecursiveDirectory searches multiple files in a directory tree.
func (s *FilesystemTestSuite) TestGrepRecursiveDirectory() {
	s.createFile("a/file1.txt", "hello from file1\n")
	s.createFile("b/file2.txt", "hello from file2\n")
	s.createFile("c/file3.txt", "no match here\n")

	res := s.callGrep(map[string]any{"path": s.testDir, "pattern": "hello"})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "hello from file1")
	s.Contains(text, "hello from file2")
	s.NotContains(text, "no match here")
}

// TestGrepIncludePatterns restricts search to matching files.
func (s *FilesystemTestSuite) TestGrepIncludePatterns() {
	s.createFile("match.go", "func main() {}\n")
	s.createFile("skip.txt", "func main() {}\n")

	res := s.callGrep(map[string]any{
		"path":            s.testDir,
		"pattern":         "func",
		"includePatterns": []string{"*.go"},
	})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "match.go")
	s.NotContains(text, "skip.txt")
}

// TestGrepExcludePatterns skips files matching exclude globs.
func (s *FilesystemTestSuite) TestGrepExcludePatterns() {
	s.createFile("keep.go", "func main() {}\n")
	s.createFile("vendor/lib.go", "func helper() {}\n")

	res := s.callGrep(map[string]any{
		"path":            s.testDir,
		"pattern":         "func",
		"excludePatterns": []string{"vendor/**"},
	})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "keep.go")
	s.NotContains(text, "vendor/lib.go")
}

// TestGrepContextLines verifies before/after context lines are returned.
func (s *FilesystemTestSuite) TestGrepContextLines() {
	f := s.createFile("ctx.txt", "line1\nline2\nTARGET\nline4\nline5\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "TARGET", "contextLines": 1})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "line2")
	s.Contains(text, "TARGET")
	s.Contains(text, "line4")
}

// TestGrepContextBeforeAfter verifies asymmetric context lines.
func (s *FilesystemTestSuite) TestGrepContextBeforeAfter() {
	f := s.createFile("asymctx.txt", "a\nb\nMATCH\nd\ne\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "MATCH", "contextBefore": 2, "contextAfter": 1})
	s.False(res.IsError)

	var result GrepResult
	jsonRes := s.callGrep(map[string]any{"path": f, "pattern": "MATCH", "contextBefore": 2, "contextAfter": 1, "format": "json"})
	s.False(jsonRes.IsError)
	err := json.Unmarshal([]byte(s.textOf(jsonRes)), &result)
	s.Require().NoError(err)
	s.Require().Len(result.Matches, 1)
	s.Len(result.Matches[0].ContextBefore, 2)
	s.Len(result.Matches[0].ContextAfter, 1)
}

// TestGrepContextOverlapNotDuplicated verifies overlapping context windows don't duplicate lines in text output.
func (s *FilesystemTestSuite) TestGrepContextLinesClampedAtFileStart() {
	f := s.createFile("clamp.txt", "TARGET\nline2\nline3\n")

	// Ask for 3 lines before, but the match is on line 1
	res := s.callGrep(map[string]any{"path": f, "pattern": "TARGET", "contextBefore": 3})
	s.False(res.IsError)

	var result GrepResult
	jsonRes := s.callGrep(map[string]any{"path": f, "pattern": "TARGET", "contextBefore": 3, "format": "json"})
	s.Require().False(jsonRes.IsError)
	err := json.Unmarshal([]byte(s.textOf(jsonRes)), &result)
	s.Require().NoError(err)
	s.Require().Len(result.Matches, 1)
	// No context before since match is at line 1
	s.Len(result.Matches[0].ContextBefore, 0)
}

// TestGrepMaxMatches caps results and sets truncated flag.
func (s *FilesystemTestSuite) TestGrepMaxMatches() {
	content := ""
	for i := 0; i < 20; i++ {
		content += "match line\n"
	}
	f := s.createFile("many.txt", content)

	res := s.callGrep(map[string]any{"path": f, "pattern": "match", "maxMatches": 5, "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal(5, result.Summary.TotalMatches)
	s.True(result.Summary.Truncated)
}

// TestGrepMaxMatchesZeroMeansUnlimited verifies 0 disables the cap.
func (s *FilesystemTestSuite) TestGrepMaxMatchesZeroMeansUnlimited() {
	content := ""
	for i := 0; i < 20; i++ {
		content += "match line\n"
	}
	f := s.createFile("unlimited.txt", content)

	res := s.callGrep(map[string]any{"path": f, "pattern": "match", "maxMatches": 0, "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal(20, result.Summary.TotalMatches)
	s.False(result.Summary.Truncated)
}

// TestGrepPathOutsideAllowed returns an error for paths outside allowed directories.
func (s *FilesystemTestSuite) TestGrepPathOutsideAllowed() {
	res := s.callGrep(map[string]any{"path": "/etc/passwd", "pattern": "root"})
	s.True(res.IsError)
	s.Contains(s.textOf(res), "access denied")
}

// TestGrepInvalidArguments returns an error for malformed JSON args.
func (s *FilesystemTestSuite) TestGrepInvalidArguments() {
	raw := []byte(`{"path": 123}`)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: ToolGrep, Arguments: raw},
	}
	res, err := s.fsServer.handleGrep(s.ctx, req)
	s.Require().NoError(err)
	s.True(res.IsError)
	s.Contains(res.Content[0].(*mcp.TextContent).Text, "Invalid arguments")
}

// TestGrepNonExistentPath returns an error when path does not exist.
func (s *FilesystemTestSuite) TestGrepNonExistentPath() {
	missing := filepath.Join(s.testDir, "ghost.txt")
	res := s.callGrep(map[string]any{"path": missing, "pattern": "x"})
	s.True(res.IsError)
}

// TestGrepJSONFormat verifies the JSON output structure.
func (s *FilesystemTestSuite) TestGrepJSONFormat() {
	f := s.createFile("json_out.txt", "apple\nbanana\napricot\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "ap", "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal("ap", result.Pattern)
	s.Equal(2, result.Summary.TotalMatches)
	s.Equal(1, result.Summary.FilesMatched)
	s.Equal(engineRE2, result.Summary.Engine)
	s.Equal(f, result.Root)
	for _, m := range result.Matches {
		s.Equal(f, m.Path)
		s.Greater(m.LineNumber, 0)
	}
}

// TestGrepTextFormat verifies ripgrep-style text output.
func (s *FilesystemTestSuite) TestGrepTextFormat() {
	f := s.createFile("text_out.go", "func main() {}\nvar x = 1\nfunc helper() {}\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "func", "format": "text"})
	s.False(res.IsError)
	text := s.textOf(res)
	// Should contain "path:linenum:text" style
	s.Contains(text, ":1:func main()")
	s.Contains(text, ":3:func helper()")
	s.NotContains(text, "var x")
}

// TestGrepEmptyFile handles empty files without errors.
func (s *FilesystemTestSuite) TestGrepEmptyFile() {
	f := s.createFile("empty.txt", "")

	res := s.callGrep(map[string]any{"path": f, "pattern": "anything"})
	s.False(res.IsError)
	s.Contains(s.textOf(res), "No matches found")
}

// TestGrepBinaryFileSkipped verifies files with null bytes are silently skipped.
func (s *FilesystemTestSuite) TestGrepBinaryFileSkipped() {
	// Create a "binary" file with null bytes
	binPath := filepath.Join(s.testDir, "binary.bin")
	binContent := []byte("some text\x00binary\x00data")
	s.Require().NoError(writeFileBytes(binPath, binContent))

	// Create a normal text file alongside it
	textFile := s.createFile("normal.txt", "hello world\n")

	res := s.callGrep(map[string]any{"path": s.testDir, "pattern": "hello", "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)

	// Should only have matched the text file
	for _, m := range result.Matches {
		s.Equal(textFile, m.Path)
	}
}

// TestGrepSingleFileSearch targets a single file directly (not a directory).
func (s *FilesystemTestSuite) TestGrepSingleFileSearch() {
	f := s.createFile("single.txt", "line A matches\nline B matches\nline C nope\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "matches", "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal(2, result.Summary.TotalMatches)
	s.Equal(1, result.Summary.FilesMatched)
	s.Equal(1, result.Summary.FilesSearched)
}

// TestGrepSummaryEngineField verifies the engine field is set in JSON output.
func (s *FilesystemTestSuite) TestGrepSummaryEngineField() {
	f := s.createFile("engine.txt", "test\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "test", "engine": "pcre2", "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal(enginePCRE2, result.Summary.Engine)
}

// TestGrepDefaultEngineIsRE2 verifies default engine when omitted.
func (s *FilesystemTestSuite) TestGrepDefaultEngineIsRE2() {
	f := s.createFile("default_engine.txt", "test\n")

	res := s.callGrep(map[string]any{"path": f, "pattern": "test", "format": "json"})
	s.False(res.IsError)

	var result GrepResult
	err := json.Unmarshal([]byte(s.textOf(res)), &result)
	s.Require().NoError(err)
	s.Equal(engineRE2, result.Summary.Engine)
}

// TestGrepTextSeparatorBetweenGroups verifies "--" separator in text output when context windows don't overlap.
func (s *FilesystemTestSuite) TestGrepTextSeparatorBetweenGroups() {
	content := "a\nb\nMATCH1\nd\ne\nf\ng\nMATCH2\ni\n"
	f := s.createFile("sep.txt", content)

	res := s.callGrep(map[string]any{"path": f, "pattern": "MATCH", "contextLines": 1, "format": "text"})
	s.False(res.IsError)
	text := s.textOf(res)
	s.Contains(text, "--")
}

// TestGrepNoSeparatorWhenWindowsAdjacent verifies no "--" when context windows touch.
func (s *FilesystemTestSuite) TestGrepNoSeparatorWhenWindowsAdjacent() {
	content := "MATCH1\nMATCH2\n"
	f := s.createFile("adj.txt", content)

	res := s.callGrep(map[string]any{"path": f, "pattern": "MATCH", "contextLines": 1, "format": "text"})
	s.False(res.IsError)
	text := s.textOf(res)
	// No "--" separator since both matches are adjacent
	s.NotContains(text, "--")
}

// ---- formatter unit tests ----

func (s *FilesystemTestSuite) TestFormatGrepTextEmpty() {
	r := &GrepResult{
		Pattern: "foo",
		Root:    "/tmp",
		Matches: nil,
		Summary: GrepSummary{TotalMatches: 0, FilesMatched: 0, FilesSearched: 1, Engine: engineRE2},
	}
	out := formatGrepText(r)
	s.Equal("No matches found", out)
}

func (s *FilesystemTestSuite) TestFormatGrepTextMatchLine() {
	r := &GrepResult{
		Pattern: "hello",
		Root:    "/tmp",
		Matches: []GrepMatch{
			{Path: "/tmp/a.txt", LineNumber: 3, LineText: "hello world"},
		},
		Summary: GrepSummary{TotalMatches: 1, FilesMatched: 1, FilesSearched: 1, Engine: engineRE2},
	}
	out := formatGrepText(r)
	s.Contains(out, "/tmp/a.txt:3:hello world")
}

func (s *FilesystemTestSuite) TestFormatGrepTextContextLines() {
	r := &GrepResult{
		Pattern: "match",
		Root:    "/tmp",
		Matches: []GrepMatch{
			{
				Path:       "/tmp/b.txt",
				LineNumber: 5,
				LineText:   "match line",
				ContextBefore: []ContextLine{
					{LineNumber: 4, LineText: "before line"},
				},
				ContextAfter: []ContextLine{
					{LineNumber: 6, LineText: "after line"},
				},
			},
		},
		Summary: GrepSummary{TotalMatches: 1, FilesMatched: 1, FilesSearched: 1, Engine: engineRE2},
	}
	out := formatGrepText(r)
	s.Contains(out, "/tmp/b.txt-4-before line")
	s.Contains(out, "/tmp/b.txt:5:match line")
	s.Contains(out, "/tmp/b.txt-6-after line")
}

func (s *FilesystemTestSuite) TestFormatGrepTextTruncated() {
	r := &GrepResult{
		Pattern: "x",
		Root:    "/tmp",
		Matches: []GrepMatch{
			{Path: "/tmp/x.txt", LineNumber: 1, LineText: "x"},
		},
		Summary: GrepSummary{TotalMatches: 1, FilesMatched: 1, FilesSearched: 5, Truncated: true, Engine: engineRE2},
	}
	out := formatGrepText(r)
	s.Contains(out, "[truncated]")
}

func (s *FilesystemTestSuite) TestFormatGrepJSONRoundtrip() {
	r := &GrepResult{
		Pattern: "test",
		Root:    "/some/path",
		Matches: []GrepMatch{
			{Path: "/some/path/f.txt", LineNumber: 2, LineText: "test me"},
		},
		Summary: GrepSummary{TotalMatches: 1, FilesMatched: 1, FilesSearched: 3, Engine: engineRE2},
	}
	out := formatGrepJSON(r)

	var decoded GrepResult
	s.Require().NoError(json.Unmarshal([]byte(out), &decoded))
	s.Equal(r.Pattern, decoded.Pattern)
	s.Equal(r.Root, decoded.Root)
	s.Equal(r.Summary.TotalMatches, decoded.Summary.TotalMatches)
	s.Equal(r.Summary.Engine, decoded.Summary.Engine)
	s.Require().Len(decoded.Matches, 1)
	s.Equal("/some/path/f.txt", decoded.Matches[0].Path)
}

// ---- helper tests ----

func (s *FilesystemTestSuite) TestPatternHasUppercase() {
	s.True(patternHasUppercase("Hello"))
	s.True(patternHasUppercase("UPPER"))
	s.True(patternHasUppercase("mixedCase"))
	s.False(patternHasUppercase("lowercase"))
	s.False(patternHasUppercase("no upper 123!"))
	s.False(patternHasUppercase(""))
}

func (s *FilesystemTestSuite) TestSearchFileBasic() {
	f := s.createFile("sf_basic.txt", "aaa\nbbb\naaa bbb\n")
	re2m := &re2Matcher{}
	var err error
	re2m.re, err = compileRE2("aaa", false)
	s.Require().NoError(err)

	matches, err := searchFile(f, re2m, 0, 0)
	s.Require().NoError(err)
	s.Len(matches, 2)
	s.Equal(1, matches[0].LineNumber)
	s.Equal(3, matches[1].LineNumber)
}

func (s *FilesystemTestSuite) TestSearchFileBinarySkipped() {
	binPath := filepath.Join(s.testDir, "sf_bin.bin")
	s.Require().NoError(writeFileBytes(binPath, []byte("text\x00binary")))

	re2m := &re2Matcher{}
	var err error
	re2m.re, err = compileRE2("text", false)
	s.Require().NoError(err)

	_, err = searchFile(binPath, re2m, 0, 0)
	s.Error(err) // should return binary file error
}

func (s *FilesystemTestSuite) TestSearchFileNotFound() {
	re2m := &re2Matcher{}
	var err error
	re2m.re, err = compileRE2("x", false)
	s.Require().NoError(err)

	_, err = searchFile("/nonexistent/path/file.txt", re2m, 0, 0)
	s.Error(err)
}

// ---- helper utilities for tests ----

func writeFileBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// compileRE2 compiles a regexp, optionally with case-insensitive flag.
func compileRE2(pattern string, ignoreCase bool) (*regexp.Regexp, error) {
	if ignoreCase {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

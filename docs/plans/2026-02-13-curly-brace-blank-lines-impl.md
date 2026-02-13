# Curly Brace Blank Line Removal Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Strip blank lines immediately after `{` and immediately before `}` in all formatted HCL blocks.

**Architecture:** Add two compiled regexes to `internal/format/format.go` that run as post-processing steps after the existing multiple-blank-line collapse. Tests follow the existing table-driven pattern in `format_test.go`.

**Tech Stack:** Go, regexp, hclwrite

---

### Task 1: Add failing tests for blank line after opening brace

**Files:**
- Modify: `internal/format/format_test.go:69` (insert new test cases before closing `}` of test table)

**Step 1: Write failing tests**

Add these test cases to the `tests` slice in `TestFormat`, before the closing `}` on line 69:

```go
		{
			name: "blank line after opening brace removed",
			input: "palette {\n\n  base = \"#191724\"\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "blank line before closing brace removed",
			input: "palette {\n  base = \"#191724\"\n\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "blank lines after and before braces both removed",
			input: "palette {\n\n  base = \"#191724\"\n\n}",
			expected: "palette {\n  base = \"#191724\"\n}",
		},
		{
			name: "nested block blank lines removed",
			input: "palette {\n\n  highlight {\n\n    low = \"#21202e\"\n\n  }\n\n}",
			expected: "palette {\n  highlight {\n    low = \"#21202e\"\n  }\n}",
		},
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/format/ -v -run TestFormat`
Expected: FAIL — the new test cases will show mismatched output since blank lines are not yet stripped.

### Task 2: Implement blank line removal

**Files:**
- Modify: `internal/format/format.go:9` (add new regex vars)
- Modify: `internal/format/format.go:20` (add new replacement steps)

**Step 1: Add the two new compiled regexes**

After line 9 (`var multipleBlankLines = ...`), add:

```go
var blankLineAfterOpenBrace = regexp.MustCompile(`\{\n\s*\n`)
var blankLineBeforeCloseBrace = regexp.MustCompile(`\n\s*\n(\s*\})`)
```

**Step 2: Apply the new regexes in the Format function**

After line 20 (`collapsed := multipleBlankLines.ReplaceAllString(...)`) and before `return`, add:

```go
	// Remove blank lines immediately after opening braces.
	collapsed = blankLineAfterOpenBrace.ReplaceAllString(collapsed, "{\n")
	// Remove blank lines immediately before closing braces.
	collapsed = blankLineBeforeCloseBrace.ReplaceAllString(collapsed, "\n${1}")
```

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/format/ -v -run TestFormat`
Expected: ALL PASS

### Task 3: Commit

**Step 1: Commit**

```bash
git add internal/format/format.go internal/format/format_test.go
git commit -m "feat(format): remove blank lines adjacent to curly braces (#64)"
```

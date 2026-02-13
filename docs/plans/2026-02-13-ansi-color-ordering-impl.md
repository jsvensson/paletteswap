# ANSI Color Ordering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** The formatter reorders ANSI block attributes into the canonical 16-color order, preserving comments.

**Architecture:** Hybrid approach — use `hclwrite.ParseConfig` to find the `ansi` block boundaries, then do text-level line grouping and reordering within those boundaries. Inserted into the `Format` pipeline between `hclwrite.Format()` and regex post-processing.

**Tech Stack:** Go, `hclwrite` (already a dependency), `internal/theme.RequiredANSIColors`

---

### Task 1: Write failing tests for ANSI reordering

**Files:**
- Modify: `internal/format/format_test.go`

**Step 1: Add test cases to the existing table-driven test**

Add these cases to the `tests` slice in `TestFormat`:

```go
{
    name: "ansi block already in order stays same",
    input: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
    expected: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
},
{
    name: "ansi block misordered gets reordered",
    input: `ansi {
  white          = palette.text
  black          = palette.overlay
  cyan           = palette.foam
  red            = palette.love
  bright_white   = palette.text
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
}
`,
    expected: `ansi {
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
},
{
    name: "ansi block comments travel with their attribute",
    input: `ansi {
  # bright colors
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
  # normal colors
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
}
`,
    expected: `ansi {
  # normal colors
  black          = palette.overlay
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text
  # bright colors
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
},
{
    name: "ansi block inline comments preserved",
    input: `ansi {
  white          = palette.text   # foreground
  black          = palette.overlay # background
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
    expected: `ansi {
  black          = palette.overlay # background
  red            = palette.love
  green          = palette.pine
  yellow         = palette.gold
  blue           = palette.foam
  magenta        = palette.iris
  cyan           = palette.foam
  white          = palette.text   # foreground
  bright_black   = palette.muted
  bright_red     = palette.love
  bright_green   = palette.pine
  bright_yellow  = palette.gold
  bright_blue    = palette.foam
  bright_magenta = palette.iris
  bright_cyan    = palette.foam
  bright_white   = palette.text
}
`,
},
{
    name: "no ansi block unchanged",
    input: `palette {
  base = "#191724"
}
`,
    expected: `palette {
  base = "#191724"
}
`,
},
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/format/ -run TestFormat -v`
Expected: New ANSI reordering tests FAIL (misordered input comes back unchanged)

**Step 3: Commit**

```
test(format): add failing tests for ANSI color ordering
```

---

### Task 2: Implement reorderANSIBlock

**Files:**
- Modify: `internal/format/format.go`

**Step 1: Add imports and the reorderANSIBlock function**

Add to imports: `"strings"`, `"github.com/hashicorp/hcl/v2"`, `"github.com/jsvensson/paletteswap/internal/theme"`

Add the `reorderANSIBlock` function:

```go
// reorderANSIBlock reorders attributes in the ansi block to canonical order.
// Uses hclwrite to find the block, then text-level reordering to preserve comments.
// Returns src unchanged if no ansi block or if parsing fails.
func reorderANSIBlock(src []byte) []byte {
	file, diags := hclwrite.ParseConfig(src, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return src
	}

	for _, block := range file.Body().Blocks() {
		if block.Type() != "ansi" {
			continue
		}

		// Find byte range of the block's inner content (between braces)
		openBrace := block.Body().Range().Start
		closeBrace := block.Body().Range().End

		inner := src[openBrace.Byte:closeBrace.Byte]
		lines := strings.Split(string(inner), "\n")

		reordered := reorderEntries(lines, theme.RequiredANSIColors)

		var result []byte
		result = append(result, src[:openBrace.Byte]...)
		result = append(result, []byte(strings.Join(reordered, "\n"))...)
		result = append(result, src[closeBrace.Byte:]...)
		return result
	}

	return src
}
```

**Step 2: Add the reorderEntries function**

```go
// reorderEntries groups lines into entries (comment lines + attribute line)
// and reorders them according to the given order.
func reorderEntries(lines []string, order []string) []string {
	type entry struct {
		name  string
		lines []string
	}

	var entries []entry
	var pending []string // accumulates comment/blank lines before an attribute

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			pending = append(pending, line)
			continue
		}
		// Attribute line: extract name (text before '=')
		if eqIdx := strings.Index(line, "="); eqIdx != -1 {
			name := strings.TrimSpace(line[:eqIdx])
			var entryLines []string
			entryLines = append(entryLines, pending...)
			entryLines = append(entryLines, line)
			entries = append(entries, entry{name: name, lines: entryLines})
			pending = nil
		} else {
			// Non-attribute, non-comment line; treat as opaque
			pending = append(pending, line)
		}
	}

	// Build order index for sorting
	orderIndex := make(map[string]int, len(order))
	for i, name := range order {
		orderIndex[name] = i
	}

	// Map entries by name
	entryMap := make(map[string]entry, len(entries))
	var unknown []entry
	for _, e := range entries {
		if _, ok := orderIndex[e.name]; ok {
			entryMap[e.name] = e
		} else {
			unknown = append(unknown, e)
		}
	}

	// Emit in canonical order
	var result []string
	for _, name := range order {
		if e, ok := entryMap[name]; ok {
			result = append(result, e.lines...)
		}
	}
	// Append any unknown entries at the end
	for _, e := range unknown {
		result = append(result, e.lines...)
	}
	// Append any trailing comments/blank lines
	result = append(result, pending...)

	return result
}
```

**Step 3: Wire into Format pipeline**

In the `Format` function, add the reorder step after `hclwrite.Format()`:

```go
func Format(content string) (string, error) {
	formatted := hclwrite.Format([]byte(content))
	// Reorder ANSI block attributes to canonical order.
	formatted = reorderANSIBlock(formatted)
	// Collapse multiple consecutive blank lines into a single blank line.
	collapsed := multipleBlankLines.ReplaceAllString(string(formatted), "\n\n")
	// Remove blank lines immediately after opening braces.
	collapsed = blankLineAfterOpenBrace.ReplaceAllString(collapsed, "{\n")
	// Remove blank lines immediately before closing braces.
	collapsed = blankLineBeforeCloseBrace.ReplaceAllString(collapsed, "\n${1}")
	return collapsed, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/format/ -run TestFormat -v`
Expected: ALL tests pass (both old and new)

**Step 5: Run full test suite**

Run: `go test ./...`
Expected: All packages pass

**Step 6: Commit**

```
feat(format): reorder ANSI block attributes to canonical order

Closes #62
```

---

### Task 3: Verify --check flag behavior

**Step 1: Create a test .pstheme file with misordered ANSI**

Write a temp file with a complete but misordered theme, then run:

```bash
go run ./cmd/paletteswap fmt --check /tmp/test.pstheme
```

Expected: exits with code 1, prints the filename

**Step 2: Run without --check to auto-fix**

```bash
go run ./cmd/paletteswap fmt /tmp/test.pstheme
```

Expected: exits 0, file is now reordered

**Step 3: Run --check again on fixed file**

```bash
go run ./cmd/paletteswap fmt --check /tmp/test.pstheme
```

Expected: exits 0, no output (file is already formatted)

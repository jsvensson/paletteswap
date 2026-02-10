# Extended Syntax Highlighting for Palette References Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Differentiate the `palette` namespace from color names in palette references (e.g., `palette.base.highlight.low` should show `palette` as namespace and `base`, `highlight`, `low` as properties), keeping everything in the LSP semantic tokens.

**Architecture:** Extend the existing semantic token system in `internal/lsp/semantic.go` by adding a `namespace` token type and modifying `extractTokensFromTraversal` to emit separate tokens for each path component instead of one token for the entire reference.

**Tech Stack:** Go, HCL parser, LSP protocol via `github.com/tliron/glsp`

---

## Current Behavior

Currently, `palette.base.highlight.low` is tokenized as a single `variable` token covering the entire string. We want:
- `palette` → `namespace` token type
- `base` → `property` token type  
- `highlight` → `property` token type
- `low` → `property` token type

---

## Task 1: Add Namespace Token Type

**Files:**
- Modify: `internal/lsp/semantic.go:10-19`

**Step 1: Add "namespace" to semantic token types**

Add "namespace" as a new token type. Insert it at index 3 (after "variable") to keep logical grouping:

```go
var semanticTokenTypes = []string{
	"keyword",    // 0: block names (meta, palette, theme, ansi, syntax)
	"property",   // 1: attribute names
	"variable",   // 2: palette references (kept for non-palette refs if needed)
	"namespace",  // 3: the "palette" namespace identifier
	"string",     // 4: hex color literals
	"function",   // 5: brighten(), darken()
	"number",     // 6: numeric literals
	"comment",    // 7: comments
}
```

**Step 2: Run existing tests to verify nothing breaks**

Run: `go test ./internal/lsp/... -v -run TestSemantic`
Expected: Tests pass (they use indices directly, so they may need updating in Task 2)

**Step 3: Commit**

```bash
git add internal/lsp/semantic.go
git commit -m "feat(lsp): add namespace token type for palette references"
```

---

## Task 2: Update Token Extraction Logic

**Files:**
- Modify: `internal/lsp/semantic.go:184-200`

**Step 1: Rewrite extractTokensFromTraversal to emit separate tokens**

Replace the current function that emits one token for the whole reference with logic that emits separate tokens for each path component:

```go
// extractTokensFromTraversal handles palette references like palette.base.highlight.low
func extractTokensFromTraversal(expr *hclsyntax.ScopeTraversalExpr, tokens []SemanticToken) []SemanticToken {
	if len(expr.Traversal) == 0 {
		return tokens
	}

	// Check if this is a palette reference (first segment is "palette")
	first, ok := expr.Traversal[0].(hcl.TraverseRoot)
	if !ok || first.Name != "palette" {
		// Not a palette reference - could tokenize as generic variable if needed
		return tokens
	}

	// Tokenize "palette" as namespace
	// The first segment's SourceRange gives us the position of "palette"
	tokens = append(tokens, SemanticToken{
		Line:      uint32(first.SrcRange.Start.Line - 1),
		StartChar: uint32(first.SrcRange.Start.Column - 1),
		Length:    uint32(len("palette")),
		Type:      tokenTypeIndices["namespace"],
		Modifiers: 0,
	})

	// Tokenize each subsequent segment as property
	for i := 1; i < len(expr.Traversal); i++ {
		switch seg := expr.Traversal[i].(type) {
		case hcl.TraverseAttr:
			tokens = append(tokens, SemanticToken{
				Line:      uint32(seg.SrcRange.Start.Line - 1),
				StartChar: uint32(seg.SrcRange.Start.Column - 1),
				Length:    uint32(len(seg.Name)),
				Type:      tokenTypeIndices["property"],
				Modifiers: 0,
			})
		case hcl.TraverseIndex:
			// Handle index access like palette.colors[0] if needed
			// For now, skip or handle as needed
		}
	}

	return tokens
}
```

**Step 2: Run tests to see what breaks**

Run: `go test ./internal/lsp/... -v -run TestSemantic`
Expected: Some tests fail because token counts changed (one token becomes multiple)

**Step 3: Commit**

```bash
git add internal/lsp/semantic.go
git commit -m "feat(lsp): tokenize palette references as namespace + properties"
```

---

## Task 3: Update Tests for New Token Structure

**Files:**
- Modify: `internal/lsp/semantic_test.go:89-104`
- Modify: `internal/lsp/semantic_test.go:130-176`

**Step 1: Update TestSemanticTokensFull_WithPaletteReference**

The test expects 5 tokens (25 integers). With the change, `palette.base` becomes 2 tokens instead of 1, so we expect 6 tokens (30 integers):

```go
func TestSemanticTokensFull_WithPaletteReference(t *testing.T) {
	content := `palette {
  base = "#191724"
}
theme {
  background = palette.base
}`
	result := semanticTokensFull(content)

	// Should have: palette(keyword), base(property),
	//              theme(keyword), background(property), 
	//              palette(namespace), base(property)  <- now 2 tokens instead of 1
	// That's 6 tokens = 30 integers
	if len(result) != 30 {
		t.Errorf("semanticTokensFull() returned %d integers, want 30", len(result))
	}
}
```

**Step 2: Update TestSemanticTokensFull_CompleteTheme expected count**

The test expects at least 80 integers (16 tokens). With nested palette references like `palette.highlight.low` becoming 3 tokens each instead of 1, the count increases:

```go
func TestSemanticTokensFull_CompleteTheme(t *testing.T) {
	// ... content stays the same ...
	
	result := semanticTokensFull(content)

	// Verify we got some tokens back
	if len(result) == 0 {
		t.Fatal("semanticTokensFull() returned empty for valid theme")
	}

	// Verify the data length is a multiple of 5
	if len(result)%5 != 0 {
		t.Errorf("semantic tokens data length %d is not a multiple of 5", len(result))
	}

	// Token count increased due to split palette references:
	// - palette.base = 2 tokens (was 1)
	// - palette.surface = 2 tokens (was 1)
	// - palette.highlight.low = 3 tokens (was 1)
	// - palette.highlight.high = 3 tokens (was 1)
	// Total increase: +6 tokens = +30 integers
	// Previous minimum was 80, new minimum is 110
	if len(result) < 110 {
		t.Errorf("semanticTokensFull() returned %d integers, expected at least 110", len(result))
	}
}
```

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/lsp/... -v -run TestSemantic`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/lsp/semantic_test.go
git commit -m "test(lsp): update tests for split palette reference tokens"
```

---

## Task 4: Add Specific Test for Nested Palette References

**Files:**
- Create test in: `internal/lsp/semantic_test.go`

**Step 1: Add test for deeply nested palette reference**

Add a new test to verify that `palette.highlight.low` produces 3 separate tokens:

```go
func TestSemanticTokensFull_NestedPaletteReference(t *testing.T) {
	content := `palette {
  highlight {
    low = "#21202e"
  }
}
theme {
  background = palette.highlight.low
}`
	result := semanticTokensFull(content)

	// Should have:
	// palette(keyword), highlight(keyword), low(property),  <- palette block
	// theme(keyword), background(property),                  <- theme block
	// palette(namespace), highlight(property), low(property) <- reference split into 3
	// That's 8 tokens = 40 integers
	if len(result) != 40 {
		t.Errorf("semanticTokensFull() returned %d integers, want 40", len(result))
	}

	// Verify the data is valid (multiple of 5)
	if len(result)%5 != 0 {
		t.Errorf("semantic tokens data length %d is not a multiple of 5", len(result))
	}
}
```

**Step 2: Run the new test**

Run: `go test ./internal/lsp/... -v -run TestSemanticTokensFull_NestedPaletteReference`
Expected: Test passes

**Step 3: Commit**

```bash
git add internal/lsp/semantic_test.go
git commit -m "test(lsp): add test for nested palette reference tokenization"
```

---

## Task 5: Run Full Test Suite

**Step 1: Run all LSP tests**

Run: `go test ./internal/lsp/... -v`
Expected: All tests pass

**Step 2: Run entire project test suite**

Run: `go test ./...`
Expected: All tests pass

**Step 3: Commit if any fixes needed**

If any fixes were needed, commit them. Otherwise, no commit needed.

---

## Task 6: Manual Verification (Optional but Recommended)

**Step 1: Build the LSP server**

Run: `go build -o pstheme-lsp ./cmd/pstheme-lsp/`
Expected: Binary builds successfully

**Step 2: Test with a sample theme file**

Create a test file `/tmp/test.pstheme`:

```hcl
palette {
  base = "#191724"
  
  highlight {
    low = "#21202e"
    high = "#524f67"
  }
}

theme {
  background = palette.base
  selection = palette.highlight.low
}
```

**Step 3: Verify in VS Code**

1. Open the test file in VS Code with your extension
2. Verify that `palette` appears with namespace highlighting (different color from `base`, `highlight`, `low`)
3. The exact colors depend on the user's theme, but you should see visual differentiation

---

## Summary of Changes

**Files Modified:**
1. `internal/lsp/semantic.go` - Added "namespace" token type, rewrote traversal extraction
2. `internal/lsp/semantic_test.go` - Updated test expectations, added nested reference test

**Behavior Change:**
- Before: `palette.base.highlight.low` = 1 token (type: variable)
- After: `palette.base.highlight.low` = 4 tokens (palette=namespace, base=property, highlight=property, low=property)

**Editor Support:**
All LSP-supporting editors (VS Code, Neovim, Zed, JetBrains) will automatically pick up these semantic tokens and apply appropriate highlighting based on the user's color theme.

---

## Rollback Plan

If issues arise, revert to single-token approach by restoring the original `extractTokensFromTraversal` function:

```go
func extractTokensFromTraversal(expr *hclsyntax.ScopeTraversalExpr, tokens []SemanticToken) []SemanticToken {
	if len(expr.Traversal) > 0 {
		first, ok := expr.Traversal[0].(hcl.TraverseRoot)
		if ok && first.Name == "palette" {
			tokens = append(tokens, SemanticToken{
				Line:      uint32(expr.SrcRange.Start.Line - 1),
				StartChar: uint32(expr.SrcRange.Start.Column - 1),
				Length:    uint32(expr.SrcRange.End.Column - expr.SrcRange.Start.Column),
				Type:      tokenTypeIndices["variable"],
				Modifiers: 0,
			})
		}
	}
	return tokens
}
```

And remove "namespace" from `semanticTokenTypes`.

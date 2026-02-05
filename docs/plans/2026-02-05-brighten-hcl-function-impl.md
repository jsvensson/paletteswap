# HCL `brighten()` Function Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `brighten(color, percentage)` function to the HCL evaluation context.

**Architecture:** Register a custom go-cty function in `buildEvalContext()` that parses the color string, calls `color.Brighten()`, and returns the result as a hex string.

**Tech Stack:** go-cty/function, existing color package

---

### Task 1: Add failing test for brighten in theme block

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add this test at the end of `config_test.go`:

```go
func TestBrightenInTheme(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten(palette.base, 0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestBrightenInTheme -v`

Expected: FAIL with error about unknown function "brighten"

---

### Task 2: Implement brighten function

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add function import**

Add to imports:

```go
"github.com/zclconf/go-cty/cty/function"
```

**Step 2: Create makeBrightenFunc**

Add this function before `buildEvalContext`:

```go
// makeBrightenFunc creates an HCL function that brightens a color.
// Usage: brighten("#hex", 0.1) or brighten(palette.color, 0.1)
func makeBrightenFunc() function.Function {
	return function.New(&function.Spec{
		Description: "Brightens a color by the given percentage (-1.0 to 1.0)",
		Params: []function.Parameter{
			{
				Name: "color",
				Type: cty.String,
			},
			{
				Name: "percentage",
				Type: cty.Number,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			colorHex := args[0].AsString()
			pct, _ := args[1].AsBigFloat().Float64()

			c, err := color.ParseHex(colorHex)
			if err != nil {
				return cty.NilVal, err
			}

			brightened := color.Brighten(c, pct)
			return cty.StringVal(brightened.Hex()), nil
		},
	})
}
```

**Step 3: Register in buildEvalContext**

Replace `buildEvalContext` with:

```go
func buildEvalContext(palette color.ColorTree) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": colorTreeToCty(palette),
		},
		Functions: map[string]function.Function{
			"brighten": makeBrightenFunc(),
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestBrightenInTheme -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add brighten() function to HCL"
```

---

### Task 3: Add test for brighten with literal hex

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the test**

```go
func TestBrightenWithLiteralHex(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten("#000000", 0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/config -run TestBrightenWithLiteralHex -v`

Expected: PASS (implementation already supports this)

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add brighten with literal hex test"
```

---

### Task 4: Add test for brighten with negative percentage (darken)

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the test**

```go
func TestBrightenNegative(t *testing.T) {
	hcl := `
palette {
  white = "#ffffff"
}

theme {
  background = brighten(palette.white, -0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	bg := theme.Theme["background"]
	if bg.Hex() != "#7f7f7f" {
		t.Errorf("Theme[background].Hex() = %q, want %q", bg.Hex(), "#7f7f7f")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/config -run TestBrightenNegative -v`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add brighten with negative percentage test"
```

---

### Task 5: Add test for brighten in ANSI block

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the test**

```go
func TestBrightenInANSI(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

ansi {
  black = brighten(palette.base, 0.5)
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	black := theme.ANSI["black"]
	if black.Hex() != "#7f7f7f" {
		t.Errorf("ANSI[black].Hex() = %q, want %q", black.Hex(), "#7f7f7f")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/config -run TestBrightenInANSI -v`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add brighten in ANSI block test"
```

---

### Task 6: Add test for brighten in syntax block

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the test**

```go
func TestBrightenInSyntax(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

syntax {
  keyword = brighten(palette.base, 0.5)
  comment {
    color  = brighten(palette.base, 0.25)
    italic = true
  }
}
`
	path := writeTempHCL(t, hcl)
	theme, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	kw := theme.Syntax["keyword"].(color.Style)
	if kw.Color.Hex() != "#7f7f7f" {
		t.Errorf("Syntax[keyword].Color.Hex() = %q, want %q", kw.Color.Hex(), "#7f7f7f")
	}
	comment := theme.Syntax["comment"].(color.Style)
	if comment.Color.Hex() != "#3f3f3f" {
		t.Errorf("Syntax[comment].Color.Hex() = %q, want %q", comment.Color.Hex(), "#3f3f3f")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/config -run TestBrightenInSyntax -v`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add brighten in syntax block test"
```

---

### Task 7: Add test for brighten error handling

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Write the test**

```go
func TestBrightenInvalidColor(t *testing.T) {
	hcl := `
palette {
  base = "#000000"
}

theme {
  background = brighten("not-a-color", 0.5)
}
`
	path := writeTempHCL(t, hcl)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid color in brighten()")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/config -run TestBrightenInvalidColor -v`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add brighten error handling test"
```

---

### Task 8: Run full test suite

**Step 1: Run all tests**

Run: `go test ./... -v`

Expected: All tests PASS

**Step 2: Final commit if any cleanup needed**

If tests pass, no action needed.

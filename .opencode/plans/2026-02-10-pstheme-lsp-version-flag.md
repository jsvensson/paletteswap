# Add --version flag to pstheme-lsp

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `--version` flag support to the `pstheme-lsp` binary that prints the version and exits, matching the behavior of the `paletteswap` binary.

**Architecture:** Parse command-line arguments in `main()` before starting the LSP server. If `--version` or `-v` is passed, print the version and exit. Otherwise, start the LSP server as usual.

**Tech Stack:** Go, standard `flag` package (already available)

---

## Task 1: Add --version flag parsing to pstheme-lsp

**Files:**
- Modify: `cmd/pstheme-lsp/main.go`

**Step 1: Write the failing test**

First, let's verify the current behavior doesn't support --version:

```bash
./pstheme-lsp --version
```

Expected: The server starts (incorrect behavior for --version)

**Step 2: Implement --version flag parsing**

Modify `cmd/pstheme-lsp/main.go` to parse flags before starting the server:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jsvensson/paletteswap/internal/lsp"
)

var version = "dev"

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version and exit (shorthand)")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	s := lsp.NewServer(version)
	if err := s.Run(); err != nil {
		os.Exit(1)
	}
}
```

**Step 3: Build and test**

```bash
go build -o pstheme-lsp ./cmd/pstheme-lsp
./pstheme-lsp --version
```

Expected output: `dev`

**Step 4: Test with version injection**

```bash
go build -ldflags "-X main.version=1.2.3" -o pstheme-lsp ./cmd/pstheme-lsp
./pstheme-lsp --version
```

Expected output: `1.2.3`

**Step 5: Verify normal operation still works**

```bash
./pstheme-lsp &
# Should start normally (no immediate output except logs)
kill %1
```

**Step 6: Commit**

```bash
git add cmd/pstheme-lsp/main.go
git commit -m "feat: add --version flag to pstheme-lsp"
```

---

## Implementation Notes

- Uses Go's standard `flag` package (no new dependencies)
- Supports both `--version` and `-v` shorthand (matching common conventions)
- Version is still passed to the LSP server for initialization handshake
- Normal LSP operation is unchanged when no flags are provided
- Exit code 0 on successful version print (standard practice)

---

**Plan complete and saved to `.opencode/plans/2026-02-10-pstheme-lsp-version-flag.md`.**

Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach would you prefer?

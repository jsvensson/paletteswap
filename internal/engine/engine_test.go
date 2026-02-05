package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/config"
)

func testTheme() *config.Theme {
	return &config.Theme{
		Meta: config.Meta{
			Name:       "Test Theme",
			Author:     "Tester",
			Appearance: "dark",
		},
		Palette: color.ColorTree{
			"base": color.Style{Color: color.Color{R: 25, G: 23, B: 36}},
			"love": color.Style{Color: color.Color{R: 235, G: 111, B: 146}},
			"highlight": color.ColorTree{
				"low":  color.Style{Color: color.Color{R: 33, G: 32, B: 46}},
				"high": color.Style{Color: color.Color{R: 82, G: 79, B: 103}},
			},
			"custom": color.ColorTree{
				"bold": color.Style{
					Color: color.Color{R: 255, G: 0, B: 0},
					Bold:  true,
				},
			},
		},
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
			"cursor":     {R: 235, G: 111, B: 146},
		},
		Syntax: color.ColorTree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
			"comment": color.Style{
				Color:  color.Color{R: 110, G: 106, B: 134},
				Italic: true,
			},
			"markup": color.ColorTree{
				"heading": color.Style{Color: color.Color{R: 235, G: 111, B: 146}},
				"bold": color.Style{
					Color: color.Color{R: 246, G: 193, B: 119},
					Bold:  true,
				},
			},
		},
		ANSI: map[string]color.Color{
			"black": {R: 25, G: 23, B: 36},
			"red":   {R: 235, G: 111, B: 146},
		},
	}
}

func setupTemplateDir(t *testing.T, templates map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range templates {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRun(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `name={{ .Meta.Name }}
bg={{ hex .Theme.background }}
cursor={{ hexBare .Theme.cursor }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	got := string(content)
	wantLines := []string{
		"name=Test Theme",
		"bg=#191724",
		"cursor=eb6f92",
	}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
}

func TestRunAppFilter(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"app1.txt.tmpl": "app1={{ .Meta.Name }}",
		"app2.txt.tmpl": "app2={{ .Meta.Name }}",
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
		Apps:         []string{"app1.txt"},
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// app1 should exist
	if _, err := os.Stat(filepath.Join(outDir, "app1.txt")); err != nil {
		t.Error("app1.txt should exist")
	}

	// app2 should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "app2.txt")); err == nil {
		t.Error("app2.txt should not exist when filtered")
	}
}

func TestRunNoTemplates(t *testing.T) {
	tmplDir := t.TempDir() // empty directory
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err == nil {
		t.Error("expected error for empty templates dir")
	}
}

func TestRunRGBFunc(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `{{ rgb .Theme.cursor }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "rgb(235, 111, 146)"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunSyntaxAccess(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `{{ $kw := index .Syntax "keyword" }}{{ hex $kw.Color }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "#31748f"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunStyleAccess(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `{{ $c := index .Syntax "comment" }}color={{ hex $c.Color }} italic={{ $c.Italic }} bold={{ $c.Bold }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "color=#6e6a86 italic=true bold=false"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunPaletteFunc(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `{{ palette "base" | hex }} {{ palette "highlight.low" | hex }} {{ palette "highlight.high" | hexBare }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "#191724 #21202e 524f67"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunStyleFunc(t *testing.T) {
	tmplDir := setupTemplateDir(t, map[string]string{
		"test.txt.tmpl": `color={{ (style "custom.bold").Color | hex }} bold={{ (style "custom.bold").Bold }}`,
	})
	outDir := filepath.Join(t.TempDir(), "output")

	e := &Engine{
		TemplatesDir: tmplDir,
		OutputDir:    outDir,
	}

	if err := e.Run(testTheme()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	want := "color=#ff0000 bold=true"
	if got := string(content); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_NewAPI(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "hex function with top-level path",
			template: `{{ hex "base" }}`,
			expected: "#191724",
		},
		{
			name:     "hex function with nested path",
			template: `{{ hex "highlight.low" }}`,
			expected: "#21202e",
		},
		{
			name:     "bhex function",
			template: `{{ bhex "base" }}`,
			expected: "191724",
		},
		{
			name:     "hexa function",
			template: `{{ hexa "base" }}`,
			expected: "#191724ff",
		},
		{
			name:     "bhexa function",
			template: `{{ bhexa "base" }}`,
			expected: "191724ff",
		},
		{
			name:     "rgb function",
			template: `{{ rgb "base" }}`,
			expected: "rgb(25, 23, 36)",
		},
		{
			name:     "rgba function",
			template: `{{ rgba "base" }}`,
			expected: "rgba(25, 23, 36, 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmplDir := setupTemplateDir(t, map[string]string{
				"test.txt.tmpl": tt.template,
			})
			outDir := filepath.Join(t.TempDir(), "output")

			e := &Engine{
				TemplatesDir: tmplDir,
				OutputDir:    outDir,
			}

			if err := e.Run(testTheme()); err != nil {
				t.Fatalf("Run() error: %v", err)
			}

			content, err := os.ReadFile(filepath.Join(outDir, "test.txt"))
			if err != nil {
				t.Fatalf("reading output: %v", err)
			}

			got := string(content)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

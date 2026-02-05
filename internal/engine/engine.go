package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/config"
)

// Engine loads and executes Go templates against a resolved Theme.
type Engine struct {
	TemplatesDir string
	OutputDir    string
	Apps         []string // if non-empty, only render these template basenames
}

// Run loads all .tmpl files from the templates directory, executes them
// with the given theme data, and writes output files.
func (e *Engine) Run(theme *config.Theme) error {
	pattern := filepath.Join(e.TemplatesDir, "*.tmpl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("globbing templates: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no .tmpl files found in %s", e.TemplatesDir)
	}

	if err := os.MkdirAll(e.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data := buildTemplateData(theme)

	for _, tmplPath := range matches {
		baseName := strings.TrimSuffix(filepath.Base(tmplPath), ".tmpl")

		if !e.shouldRender(baseName) {
			continue
		}

		if err := e.renderTemplate(tmplPath, baseName, data); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) shouldRender(name string) bool {
	// If no apps are specified, render all.
	if len(e.Apps) == 0 {
		return true
	}

	return slices.Contains(e.Apps, name)
}

func (e *Engine) renderTemplate(tmplPath, outputName string, data templateData) error {
	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(data.FuncMap).ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", tmplPath, err)
	}

	outPath := filepath.Join(e.OutputDir, outputName)
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template %s: %w", tmplPath, err)
	}

	return nil
}

// templateData is the data passed to templates.
type templateData struct {
	Meta    config.Meta
	Palette color.ColorTree
	Theme   map[string]color.Color
	Syntax  color.ColorTree
	ANSI    map[string]color.Color
	FuncMap template.FuncMap
}

// resolveColorPath resolves a universal dot-notation path to a Color.
// Supports paths like "palette.base", "theme.background", "ansi.black", "syntax.keyword".
func resolveColorPath(path string, data templateData) (color.Color, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return color.Color{}, fmt.Errorf("invalid path %q: must be block.name format", path)
	}

	block := parts[0]
	rest := parts[1:]

	switch block {
	case "palette":
		style := getStyleFromTree(data.Palette, rest)
		if style.Color == (color.Color{}) {
			return color.Color{}, fmt.Errorf("palette path not found: %s", path)
		}
		return style.Color, nil

	case "theme":
		if len(rest) != 1 {
			return color.Color{}, fmt.Errorf("theme paths must be single-level: %s", path)
		}
		c, ok := data.Theme[rest[0]]
		if !ok {
			return color.Color{}, fmt.Errorf("theme color not found: %s", rest[0])
		}
		return c, nil

	case "ansi":
		if len(rest) != 1 {
			return color.Color{}, fmt.Errorf("ansi paths must be single-level: %s", path)
		}
		c, ok := data.ANSI[rest[0]]
		if !ok {
			return color.Color{}, fmt.Errorf("ansi color not found: %s", rest[0])
		}
		return c, nil

	case "syntax":
		style := getStyleFromTree(data.Syntax, rest)
		if style.Color == (color.Color{}) {
			return color.Color{}, fmt.Errorf("syntax path not found: %s", path)
		}
		return style.Color, nil

	default:
		return color.Color{}, fmt.Errorf("unknown block %q (valid: palette, theme, ansi, syntax)", block)
	}
}

// getStyleFromTree traverses a ColorTree using path segments and returns the Style.
func getStyleFromTree(tree color.ColorTree, path []string) color.Style {
	if len(path) == 0 {
		return color.Style{}
	}

	current := tree
	for i, part := range path {
		val, ok := current[part]
		if !ok {
			return color.Style{}
		}

		// Last part should be a Style
		if i == len(path)-1 {
			if style, ok := val.(color.Style); ok {
				return style
			}
			return color.Style{}
		}

		// Intermediate parts should be ColorTrees
		if subtree, ok := val.(color.ColorTree); ok {
			current = subtree
		} else {
			return color.Style{}
		}
	}

	return color.Style{}
}

func buildTemplateData(theme *config.Theme) templateData {
	return templateData{
		Meta:    theme.Meta,
		Palette: theme.Palette,
		Theme:   theme.Theme,
		Syntax:  theme.Syntax,
		ANSI:    theme.ANSI,
		FuncMap: template.FuncMap{
			"hex": func(c color.Color) string {
				return c.Hex()
			},
			"hexBare": func(c color.Color) string {
				return c.HexBare()
			},
			"rgb": func(c color.Color) string {
				return c.RGB()
			},
			"palette": func(path string) color.Color {
				return getStyleFromPath(theme.Palette, path).Color
			},
			"style": func(path string) color.Style {
				return getStyleFromPath(theme.Palette, path)
			},
		},
	}
}

// getStyleFromPath traverses a ColorTree using a dot-separated path
// and returns the Style at that path. Returns empty Style if not found.
func getStyleFromPath(tree color.ColorTree, path string) color.Style {
	parts := strings.Split(path, ".")
	current := tree

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return color.Style{}
		}

		// Last part should be a Style
		if i == len(parts)-1 {
			if style, ok := val.(color.Style); ok {
				return style
			}
			return color.Style{}
		}

		// Intermediate parts should be ColorTrees
		if subtree, ok := val.(color.ColorTree); ok {
			current = subtree
		} else {
			return color.Style{}
		}
	}

	return color.Style{}
}

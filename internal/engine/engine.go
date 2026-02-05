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

func buildTemplateData(theme *config.Theme) templateData {
	return templateData{
		Meta:    theme.Meta,
		Palette: theme.Palette,
		Theme:   theme.Theme,
		Syntax:  theme.Syntax,
		ANSI:    theme.ANSI,
		FuncMap: template.FuncMap{
			"hex": func(v any) (string, error) {
				switch val := v.(type) {
				case string:
					style, err := getStyleFromPathWithError(theme.Palette, val)
					if err != nil {
						return "", err
					}
					return style.Color.Hex(), nil
				case color.Color:
					return val.Hex(), nil
				default:
					return "", fmt.Errorf("hex: expected string path or Color, got %T", v)
				}
			},
			"hexBare": func(c color.Color) string {
				return c.HexBare()
			},
			"bhex": func(path string) (string, error) {
				style, err := getStyleFromPathWithError(theme.Palette, path)
				if err != nil {
					return "", err
				}
				return style.Color.HexBare(), nil
			},
			"hexa": func(path string) (string, error) {
				style, err := getStyleFromPathWithError(theme.Palette, path)
				if err != nil {
					return "", err
				}
				return style.Color.HexAlpha(), nil
			},
			"bhexa": func(path string) (string, error) {
				style, err := getStyleFromPathWithError(theme.Palette, path)
				if err != nil {
					return "", err
				}
				return style.Color.HexBareAlpha(), nil
			},
			"rgb": func(v any) (string, error) {
				switch val := v.(type) {
				case string:
					style, err := getStyleFromPathWithError(theme.Palette, val)
					if err != nil {
						return "", err
					}
					return style.Color.RGB(), nil
				case color.Color:
					return val.RGB(), nil
				default:
					return "", fmt.Errorf("rgb: expected string path or Color, got %T", v)
				}
			},
			"rgba": func(path string) (string, error) {
				style, err := getStyleFromPathWithError(theme.Palette, path)
				if err != nil {
					return "", err
				}
				return style.Color.RGBA(), nil
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

// getStyleFromPathWithError traverses a ColorTree using dot-separated path and returns error if not found
func getStyleFromPathWithError(tree color.ColorTree, path string) (color.Style, error) {
	parts := strings.Split(path, ".")
	current := tree

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return color.Style{}, fmt.Errorf("palette path not found: %s", path)
		}

		// If this is the last part, we expect a Style
		if i == len(parts)-1 {
			if style, ok := value.(color.Style); ok {
				return style, nil
			}
			return color.Style{}, fmt.Errorf("palette path %s is not a color", path)
		}

		// Otherwise, we expect a ColorTree to continue traversing
		if subtree, ok := value.(color.ColorTree); ok {
			current = subtree
		} else {
			return color.Style{}, fmt.Errorf("palette path %s: expected subtree at %s", path, part)
		}
	}

	return color.Style{}, fmt.Errorf("palette path not found: %s", path)
}

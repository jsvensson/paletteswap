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
	Palette color.Tree
	Theme   map[string]color.Color
	Syntax  color.Tree
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

// getStyleFromTree traverses a Tree using path segments and returns the Style.
func getStyleFromTree(tree color.Tree, path []string) color.Style {
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

		// Intermediate parts should be Trees
		if subtree, ok := val.(color.Tree); ok {
			current = subtree
		} else {
			return color.Style{}
		}
	}

	return color.Style{}
}

func buildTemplateData(theme *config.Theme) templateData {
	data := templateData{
		Meta:    theme.Meta,
		Palette: theme.Palette,
		Theme:   theme.Theme,
		Syntax:  theme.Syntax,
		ANSI:    theme.ANSI,
	}

	// Universal path-based functions
	data.FuncMap = template.FuncMap{
		"hex": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.Hex(), nil
			case color.Color:
				return v.Hex(), nil
			default:
				return "", fmt.Errorf("hex: unsupported type %T", arg)
			}
		},
		"bhex": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.HexBare(), nil
			case color.Color:
				return v.HexBare(), nil
			default:
				return "", fmt.Errorf("bhex: unsupported type %T", arg)
			}
		},
		"hexa": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.HexAlpha(), nil
			case color.Color:
				return v.HexAlpha(), nil
			default:
				return "", fmt.Errorf("hexa: unsupported type %T", arg)
			}
		},
		"bhexa": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.HexBareAlpha(), nil
			case color.Color:
				return v.HexBareAlpha(), nil
			default:
				return "", fmt.Errorf("bhexa: unsupported type %T", arg)
			}
		},
		"rgb": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.RGB(), nil
			case color.Color:
				return v.RGB(), nil
			default:
				return "", fmt.Errorf("rgb: unsupported type %T", arg)
			}
		},
		"rgba": func(arg any) (string, error) {
			switch v := arg.(type) {
			case string:
				c, err := resolveColorPath(v, data)
				if err != nil {
					return "", err
				}
				return c.RGBA(), nil
			case color.Color:
				return v.RGBA(), nil
			default:
				return "", fmt.Errorf("rgba: unsupported type %T", arg)
			}
		},
		"style": func(path string) (color.Style, error) {
			parts := strings.Split(path, ".")
			if len(parts) < 2 {
				return color.Style{}, fmt.Errorf("invalid path %q", path)
			}

			block := parts[0]
			rest := parts[1:]

			switch block {
			case "palette":
				return getStyleFromTree(data.Palette, rest), nil
			case "syntax":
				return getStyleFromTree(data.Syntax, rest), nil
			default:
				return color.Style{}, fmt.Errorf("style only supports palette/syntax blocks, got %q", block)
			}
		},
	}

	return data
}


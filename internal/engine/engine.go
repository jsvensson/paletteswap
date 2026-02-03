package engine

import (
	"fmt"
	"os"
	"path/filepath"
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
	funcMap := buildFuncMap()

	for _, tmplPath := range matches {
		baseName := strings.TrimSuffix(filepath.Base(tmplPath), ".tmpl")

		if !e.shouldRender(baseName) {
			continue
		}

		if err := e.renderTemplate(tmplPath, baseName, data, funcMap); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) shouldRender(name string) bool {
	if len(e.Apps) == 0 {
		return true
	}
	for _, app := range e.Apps {
		if app == name {
			return true
		}
	}
	return false
}

func (e *Engine) renderTemplate(tmplPath, outputName string, data templateData, funcMap template.FuncMap) error {
	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(funcMap).ParseFiles(tmplPath)
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
	Palette map[string]color.Color
	Theme   map[string]color.Color
	Syntax  color.ColorTree
	ANSI    map[string]color.Color
}

func buildTemplateData(theme *config.Theme) templateData {
	return templateData{
		Meta:    theme.Meta,
		Palette: theme.Palette,
		Theme:   theme.Theme,
		Syntax:  theme.Syntax,
		ANSI:    theme.ANSI,
	}
}

func buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"hex": func(c color.Color) string {
			return c.Hex()
		},
		"hexBare": func(c color.Color) string {
			return c.HexBare()
		},
		"rgb": func(c color.Color) string {
			return c.RGB()
		},
	}
}

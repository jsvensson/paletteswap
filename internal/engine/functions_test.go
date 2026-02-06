package engine

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/jsvensson/paletteswap/internal/config"
)

func TestTemplateFunctions_Hex(t *testing.T) {
	theme := &config.Theme{
		Palette: color.Tree{
			"base": color.Style{Color: color.Color{R: 25, G: 23, B: 36}},
		},
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
		ANSI: map[string]color.Color{
			"black": {R: 0, G: 0, B: 0},
		},
		Syntax: color.Tree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
		},
	}

	data := buildTemplateData(theme)

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"palette path", `{{ hex "palette.base" }}`, "#191724"},
		{"theme path", `{{ hex "theme.background" }}`, "#191724"},
		{"ansi path", `{{ hex "ansi.black" }}`, "#000000"},
		{"syntax path", `{{ hex "syntax.keyword" }}`, "#31748f"},
		{"direct field", `{{ hex .Theme.background }}`, "#191724"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute error: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateFunctions_Bhex(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ bhex "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "191724"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Hexa(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ hexa "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "#191724ff"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Bhexa(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ bhexa "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "191724ff"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_RGB(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tests := []struct {
		template string
		want     string
	}{
		{`{{ rgb "theme.background" }}`, "rgb(25, 23, 36)"},
		{`{{ rgb .Theme.background }}`, "rgb(25, 23, 36)"},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(tt.template)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute error: %v", err)
			}

			got := strings.TrimSpace(buf.String())
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateFunctions_RGBA(t *testing.T) {
	theme := &config.Theme{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(`{{ rgba "theme.background" }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "rgba(25, 23, 36, 1.0)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateFunctions_Style(t *testing.T) {
	theme := &config.Theme{
		Syntax: color.Tree{
			"keyword": color.Style{
				Color: color.Color{R: 49, G: 116, B: 143},
				Bold:  true,
			},
		},
	}

	data := buildTemplateData(theme)

	tmpl, err := template.New("test").Funcs(data.FuncMap).Parse(
		`{{ $s := style "syntax.keyword" }}{{ if $s.Bold }}bold{{ end }}`,
	)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	want := "bold"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

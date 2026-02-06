package paletteswap

import (
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
)

func TestResolveColorPath_Palette(t *testing.T) {
	data := templateData{
		Palette: color.Tree{
			"base": color.Style{Color: color.Color{R: 25, G: 23, B: 36}},
			"highlight": color.Tree{
				"low": color.Style{Color: color.Color{R: 33, G: 32, B: 46}},
			},
		},
	}

	tests := []struct {
		path string
		want color.Color
	}{
		{"palette.base", color.Color{R: 25, G: 23, B: 36}},
		{"palette.highlight.low", color.Color{R: 33, G: 32, B: 46}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := resolveColorPath(tt.path, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveColorPath_Theme(t *testing.T) {
	data := templateData{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
			"foreground": {R: 224, G: 222, B: 244},
		},
	}

	got, err := resolveColorPath("theme.background", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := color.Color{R: 25, G: 23, B: 36}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveColorPath_ANSI(t *testing.T) {
	data := templateData{
		ANSI: map[string]color.Color{
			"black":        {R: 0, G: 0, B: 0},
			"bright_black": {R: 128, G: 128, B: 128},
		},
	}

	tests := []struct {
		path string
		want color.Color
	}{
		{"ansi.black", color.Color{R: 0, G: 0, B: 0}},
		{"ansi.bright_black", color.Color{R: 128, G: 128, B: 128}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := resolveColorPath(tt.path, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveColorPath_Syntax(t *testing.T) {
	data := templateData{
		Syntax: color.Tree{
			"keyword": color.Style{Color: color.Color{R: 49, G: 116, B: 143}},
		},
	}

	got, err := resolveColorPath("syntax.keyword", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := color.Color{R: 49, G: 116, B: 143}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveColorPath_InvalidBlock(t *testing.T) {
	data := templateData{}

	_, err := resolveColorPath("invalid.path", data)
	if err == nil {
		t.Fatal("expected error for invalid block, got nil")
	}
}

func TestResolveColorPath_PathNotFound(t *testing.T) {
	data := templateData{
		Theme: map[string]color.Color{
			"background": {R: 25, G: 23, B: 36},
		},
	}

	_, err := resolveColorPath("theme.notfound", data)
	if err == nil {
		t.Fatal("expected error for path not found, got nil")
	}
}

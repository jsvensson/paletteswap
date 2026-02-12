package theme

import (
	"testing"

	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeToCty_Leaf(t *testing.T) {
	c, _ := color.ParseHex("#ff0000")
	node := &color.Node{Color: &c}
	val := NodeToCty(node)
	if val.Type() != cty.String {
		t.Fatalf("expected string, got %s", val.Type().FriendlyName())
	}
	if val.AsString() != "#ff0000" {
		t.Errorf("got %q, want %q", val.AsString(), "#ff0000")
	}
}

func TestNodeToCty_NamespaceOnly(t *testing.T) {
	low, _ := color.ParseHex("#21202e")
	node := &color.Node{
		Children: map[string]*color.Node{
			"low": {Color: &low},
		},
	}
	val := NodeToCty(node)
	if !val.Type().IsObjectType() {
		t.Fatalf("expected object, got %s", val.Type().FriendlyName())
	}
	if val.GetAttr("low").AsString() != "#21202e" {
		t.Errorf("low = %q, want %q", val.GetAttr("low").AsString(), "#21202e")
	}
}

func TestNodeToCty_ColorAndChildren(t *testing.T) {
	gray, _ := color.ParseHex("#c0c0c0")
	low, _ := color.ParseHex("#21202e")
	node := &color.Node{
		Color: &gray,
		Children: map[string]*color.Node{
			"low": {Color: &low},
		},
	}
	val := NodeToCty(node)
	if !val.Type().IsObjectType() {
		t.Fatalf("expected object, got %s", val.Type().FriendlyName())
	}
	if val.GetAttr("color").AsString() != "#c0c0c0" {
		t.Errorf("color = %q, want %q", val.GetAttr("color").AsString(), "#c0c0c0")
	}
	if val.GetAttr("low").AsString() != "#21202e" {
		t.Errorf("low = %q, want %q", val.GetAttr("low").AsString(), "#21202e")
	}
}

func TestResolveColor_String(t *testing.T) {
	val := cty.StringVal("#ff0000")
	got, err := ResolveColor(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "#ff0000" {
		t.Errorf("got %q, want %q", got, "#ff0000")
	}
}

func TestResolveColor_ObjectWithColor(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"color": cty.StringVal("#c0c0c0"),
		"low":   cty.StringVal("#21202e"),
	})
	got, err := ResolveColor(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "#c0c0c0" {
		t.Errorf("got %q, want %q", got, "#c0c0c0")
	}
}

func TestResolveColor_ObjectWithoutColor(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"low": cty.StringVal("#21202e"),
	})
	_, err := ResolveColor(val)
	if err == nil {
		t.Fatal("expected error for object without color key")
	}
}

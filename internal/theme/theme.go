package theme

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/jsvensson/paletteswap/internal/color"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// RequiredANSIColors defines the 16 standard terminal colors that must be present.
var RequiredANSIColors = []string{
	"black", "red", "green", "yellow",
	"blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}

// ResolveColor extracts a color hex string from a cty.Value.
// If the value is a string, return it directly.
// If the value is an object, extract the "color" key.
func ResolveColor(val cty.Value) (string, error) {
	if val.Type() == cty.String {
		return val.AsString(), nil
	}
	if val.Type().IsObjectType() {
		if val.Type().HasAttribute("color") {
			colorVal := val.GetAttr("color")
			if colorVal.Type() == cty.String {
				return colorVal.AsString(), nil
			}
		}
		return "", fmt.Errorf("object has no 'color' attribute; reference a specific child or add a color attribute")
	}
	return "", fmt.Errorf("expected string or object with color attribute, got %s", val.Type().FriendlyName())
}

// NodeToCty converts a color.Node to a cty.Value for HCL evaluation context.
// Leaf nodes (no children) become cty.StringVal.
// Nodes with children become cty.ObjectVal, with "color" as a sibling key if the node has its own color.
func NodeToCty(node *color.Node) cty.Value {
	if node.Children == nil {
		// Leaf node: just a color string
		if node.Color != nil {
			return cty.StringVal(node.Color.Hex())
		}
		// Namespace-only leaf with no children — shouldn't happen, but handle gracefully
		return cty.EmptyObjectVal
	}

	vals := make(map[string]cty.Value, len(node.Children)+1)

	// Add the block's own color as "color" key
	if node.Color != nil {
		vals["color"] = cty.StringVal(node.Color.Hex())
	}

	// Add children
	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals[k] = NodeToCty(node.Children[k])
	}

	return cty.ObjectVal(vals)
}

// MakeBrightenFunc creates an HCL function that brightens a color.
// Usage: brighten("#hex", 0.1) or brighten(palette.color, 0.1)
func MakeBrightenFunc() function.Function {
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

// MakeDarkenFunc creates an HCL function that darkens a color.
// Usage: darken("#hex", 0.1) or darken(palette.color, 0.1)
func MakeDarkenFunc() function.Function {
	return function.New(&function.Spec{
		Description: "Darkens a color by the given percentage (0.0 to 1.0)",
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

			darkened := color.Darken(c, pct)
			return cty.StringVal(darkened.Hex()), nil
		},
	})
}

// BuildEvalContext creates an HCL evaluation context with palette variables
// and brighten/darken functions.
func BuildEvalContext(palette *color.Node) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"palette": NodeToCty(palette),
		},
		Functions: map[string]function.Function{
			"brighten": MakeBrightenFunc(),
			"darken":   MakeDarkenFunc(),
		},
	}
}

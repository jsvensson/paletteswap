package color

import (
	"testing"
)

func TestParseHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Color
		wantErr bool
	}{
		{"with hash", "#eb6f92", Color{235, 111, 146}, false},
		{"without hash", "eb6f92", Color{235, 111, 146}, false},
		{"black", "#000000", Color{0, 0, 0}, false},
		{"white", "#ffffff", Color{255, 255, 255}, false},
		{"uppercase", "#AABBCC", Color{170, 187, 204}, false},
		{"too short", "#fff", Color{}, true},
		{"too long", "#aabbccdd", Color{}, true},
		{"invalid chars", "#zzzzzz", Color{}, true},
		{"empty", "", Color{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHex(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseHex(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestColorHex(t *testing.T) {
	c := Color{235, 111, 146}
	want := "#eb6f92"
	if got := c.Hex(); got != want {
		t.Errorf("Color.Hex() = %q, want %q", got, want)
	}
}

func TestColorHexBare(t *testing.T) {
	c := Color{235, 111, 146}
	want := "eb6f92"
	if got := c.HexBare(); got != want {
		t.Errorf("Color.HexBare() = %q, want %q", got, want)
	}
}

func TestColorRGB(t *testing.T) {
	c := Color{235, 111, 146}
	want := "rgb(235, 111, 146)"
	if got := c.RGB(); got != want {
		t.Errorf("Color.RGB() = %q, want %q", got, want)
	}
}

func TestColorHexZeroPadding(t *testing.T) {
	c := Color{0, 5, 10}
	want := "#00050a"
	if got := c.Hex(); got != want {
		t.Errorf("Color.Hex() = %q, want %q", got, want)
	}
}

func TestBrighten(t *testing.T) {
	tests := []struct {
		name       string
		color      Color
		percentage float64
		want       Color
	}{
		{
			name:       "brighten red by 10%",
			color:      Color{255, 0, 0},
			percentage: 0.1,
			want:       Color{255, 50, 50},
		},
		{
			name:       "brighten gray by 20%",
			color:      Color{128, 128, 128},
			percentage: 0.2,
			want:       Color{179, 179, 179},
		},
		{
			name:       "white stays white",
			color:      Color{255, 255, 255},
			percentage: 0.5,
			want:       Color{255, 255, 255},
		},
		{
			name:       "brighten black by 50%",
			color:      Color{0, 0, 0},
			percentage: 0.5,
			want:       Color{127, 127, 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Brighten(tt.color, tt.percentage)
			if got != tt.want {
				t.Errorf("Brighten(%v, %v) = %v, want %v", tt.color, tt.percentage, got, tt.want)
			}
		})
	}
}

func TestDarken(t *testing.T) {
	tests := []struct {
		name       string
		color      Color
		percentage float64
		want       Color
	}{
		{
			name:       "darken red by 10%",
			color:      Color{255, 0, 0},
			percentage: 0.1,
			want:       Color{204, 0, 0},
		},
		{
			name:       "darken gray by 20%",
			color:      Color{128, 128, 128},
			percentage: 0.2,
			want:       Color{77, 77, 77},
		},
		{
			name:       "darken blue by 10%",
			color:      Color{0, 0, 255},
			percentage: 0.1,
			want:       Color{0, 0, 204},
		},
		{
			name:       "black stays black",
			color:      Color{0, 0, 0},
			percentage: 0.5,
			want:       Color{0, 0, 0},
		},
		{
			name:       "darken white by 50%",
			color:      Color{255, 255, 255},
			percentage: 0.5,
			want:       Color{127, 127, 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Darken(tt.color, tt.percentage)
			if got != tt.want {
				t.Errorf("Darken(%v, %v) = %v, want %v", tt.color, tt.percentage, got, tt.want)
			}
		})
	}
}

func TestColor_RGBA(t *testing.T) {
	tests := []struct {
		name     string
		color    Color
		expected string
	}{
		{
			name:     "red with full opacity",
			color:    Color{255, 0, 0},
			expected: "rgba(255, 0, 0, 1.0)",
		},
		{
			name:     "green with full opacity",
			color:    Color{0, 255, 0},
			expected: "rgba(0, 255, 0, 1.0)",
		},
		{
			name:     "dark color",
			color:    Color{25, 23, 36},
			expected: "rgba(25, 23, 36, 1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.RGBA()
			if got != tt.expected {
				t.Errorf("RGBA() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestColor_HexAlpha(t *testing.T) {
	tests := []struct {
		name     string
		color    Color
		expected string
	}{
		{
			name:     "red with full opacity",
			color:    Color{255, 0, 0},
			expected: "#ff0000ff",
		},
		{
			name:     "dark color",
			color:    Color{25, 23, 36},
			expected: "#191724ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.HexAlpha()
			if got != tt.expected {
				t.Errorf("HexAlpha() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNode_Lookup(t *testing.T) {
	// Build: palette { black = "#000000"; highlight { color = "#c0c0c0"; low = "#21202e" } }
	black, _ := ParseHex("#000000")
	gray, _ := ParseHex("#c0c0c0")
	low, _ := ParseHex("#21202e")

	root := &Node{
		Children: map[string]*Node{
			"black": {Color: &black},
			"highlight": {
				Color: &gray,
				Children: map[string]*Node{
					"low": {Color: &low},
				},
			},
		},
	}

	tests := []struct {
		name    string
		path    []string
		want    string
		wantErr bool
	}{
		{"flat leaf", []string{"black"}, "#000000", false},
		{"nested block with color", []string{"highlight"}, "#c0c0c0", false},
		{"nested child", []string{"highlight", "low"}, "#21202e", false},
		{"not found", []string{"missing"}, "", true},
		{"namespace only", []string{"nocolor"}, "", true},
	}

	// Add a namespace-only node for the error case
	root.Children["nocolor"] = &Node{
		Children: map[string]*Node{
			"child": {Color: &black},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := root.Lookup(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup(%v) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if err == nil && got.Hex() != tt.want {
				t.Errorf("Lookup(%v) = %q, want %q", tt.path, got.Hex(), tt.want)
			}
		})
	}
}

func TestApplyLightnessSteps_FlatLeaf(t *testing.T) {
	c, _ := ParseHex("#808080")
	root := &Node{
		Children: map[string]*Node{
			"gray": {Color: &c},
		},
	}

	ApplyLightnessSteps(root, 0.3, 0.9, 3)

	if root.Children["gray"].Color == nil {
		t.Fatal("expected gray to retain its color")
	}
	if root.Children["gray"].Color.Hex() != "#808080" {
		t.Errorf("gray.Color = %q, want %q", root.Children["gray"].Color.Hex(), "#808080")
	}

	if root.Children["gray"].Children == nil {
		t.Fatal("expected gray to have children after stepping")
	}
	for _, name := range []string{"l1", "l2", "l3"} {
		child, ok := root.Children["gray"].Children[name]
		if !ok {
			t.Errorf("expected child %q", name)
			continue
		}
		if child.Color == nil {
			t.Errorf("%s has nil color", name)
		}
	}
}

func TestApplyLightnessSteps_Nested(t *testing.T) {
	mid, _ := ParseHex("#403d52")
	root := &Node{
		Children: map[string]*Node{
			"highlight": {
				Children: map[string]*Node{
					"mid": {Color: &mid},
				},
			},
		},
	}

	ApplyLightnessSteps(root, 0.4, 0.8, 2)

	midNode := root.Children["highlight"].Children["mid"]
	if midNode.Children == nil {
		t.Fatal("expected mid to have children")
	}
	if _, ok := midNode.Children["l1"]; !ok {
		t.Error("expected l1")
	}
	if _, ok := midNode.Children["l2"]; !ok {
		t.Error("expected l2")
	}
}

func TestApplyLightnessSteps_PreservesOriginalColor(t *testing.T) {
	c, _ := ParseHex("#eb6f92")
	root := &Node{
		Children: map[string]*Node{
			"love": {Color: &c},
		},
	}

	ApplyLightnessSteps(root, 0.5, 0.9, 3)

	got, err := root.Lookup([]string{"love"})
	if err != nil {
		t.Fatalf("Lookup(love) error: %v", err)
	}
	if got.Hex() != "#eb6f92" {
		t.Errorf("love = %q, want %q", got.Hex(), "#eb6f92")
	}
}

func TestApplyLightnessSteps_SkipsNamespaceOnly(t *testing.T) {
	child, _ := ParseHex("#000000")
	root := &Node{
		Children: map[string]*Node{
			"group": {
				Children: map[string]*Node{
					"inner": {Color: &child},
				},
			},
		},
	}

	ApplyLightnessSteps(root, 0.3, 0.9, 3)

	if _, ok := root.Children["group"].Children["l1"]; ok {
		t.Error("namespace-only group should not get lightness steps")
	}
	if root.Children["group"].Children["inner"].Children == nil {
		t.Fatal("expected inner to have children")
	}
	if _, ok := root.Children["group"].Children["inner"].Children["l1"]; !ok {
		t.Error("expected inner.l1")
	}
}

func TestColor_HexBareAlpha(t *testing.T) {
	tests := []struct {
		name     string
		color    Color
		expected string
	}{
		{
			name:     "red with full opacity",
			color:    Color{255, 0, 0},
			expected: "ff0000ff",
		},
		{
			name:     "dark color",
			color:    Color{25, 23, 36},
			expected: "191724ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.HexBareAlpha()
			if got != tt.expected {
				t.Errorf("HexBareAlpha() = %v, want %v", got, tt.expected)
			}
		})
	}
}

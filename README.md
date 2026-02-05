# PaletteSwap

PaletteSwap generates application-specific color themes from a single theme source file. Define your colors once, then transform them into config files for any terminal, editor, or tool that supports custom themes written in plain text formats.

## HCL Theme Format

> [!WARNING]
> PaletteSwap is still in early development. The theme and templates formats are subject to breaking changes. Version 0.2.0 introduced a breaking change to the template API - see migration guide below.

### Migration from v0.1.x to v0.2.x

The template API has been redesigned for clarity. Update your custom templates:

**Old syntax (v0.1.x):**
```
{{ palette "highlight.low" | hex }}
{{ palette "base" | hexBare }}
```

**New syntax (v0.2.x):**
```
{{ hex "highlight.low" }}
{{ bhex "base" }}
```

**Conversion rules:**
- `palette "X" | hex` → `hex "X"`
- `palette "X" | hexBare` → `bhex "X"`
- `palette "X" | rgb` → `rgb "X"`
- Direct field access unchanged: `{{ hexBare .Theme.background }}` → `{{ bhex .Theme.background }}`

Themes are defined in HCL with a clear, hierarchical structure:

### Meta Block

Contains theme metadata:

```hcl
meta {
  name       = "Rosé Pine"
  author     = "Rosé Pine"
  appearance = "dark"  # or "light"
}
```

### Palette Block

Define your color constants as hex values. The names can be arbitrary. Supports nested blocks for organizing colors hierarchically.

```hcl
palette {
  base    = "#191724"
  surface = "#1f1d2e"
  text    = "#e0def4"
  love    = "#eb6f92"
  gold    = "#f6c177"

  highlight {
    low  = "#21202e"
    mid  = "#403d52"
    high = "#524f67"
  }
}
```

Palette colors can be referenced by other blocks using `palette.<name>` syntax for direct colors, or `palette.<scope>.<name>` for nested colors.

All palette values are accessible in templates using color formatting functions with dot-notation paths:

```text
{{ hex "highlight.low" }}
{{ bhex "base" }}
```

To access style flags (bold, italic, underline), use the `style` function:

```text
{{ if (style "custom.bold").Bold }}bold{{ end }}
```

### HCL Functions

#### brighten()

The `brighten(color, percentage)` function creates lighter or darker variations of colors by adjusting lightness in HSL color space.

```hcl
palette {
  base = "#191724"
  surface = brighten(base, 0.1)        # 10% lighter
  overlay = brighten(base, -0.1)       # 10% darker (negative percentage)
}

theme {
  background = palette.base
  surface    = brighten(palette.base, 0.05)   # derive from palette
  highlight  = brighten("#ffffff", -0.2)      # or use literal hex
}
```

Parameters:
- `color` - hex string (e.g., `"#191724"`) or palette reference (e.g., `base` or `palette.highlight.low`)
- `percentage` - float value where positive brightens and negative darkens (typically -1.0 to 1.0)

The function works in all HCL blocks: `palette`, `theme`, `ansi`, and `syntax`.

### Theme Block

Maps palette colors to UI elements:

```hcl
theme {
  background = palette.base
  foreground = palette.text
  cursor     = palette.highlight.high
  selection  = palette.highlight.mid
}
```

### ANSI Block

Standard 16-color terminal palette:

```hcl
ansi {
  black   = palette.overlay
  red     = palette.love
  green   = palette.pine
  yellow  = palette.gold
  # ... bright variants
}
```

### Syntax Block

> [!WARNING]
> The syntax names can currently be entirely arbitrary; there is no fixed standard yet. This should be decided upon before PaletteSwap can reliably be adopted. See [issue #5](https://github.com/jsvensson/paletteswap/issues/5).

Code highlighting with optional styling:

```hcl
syntax {
  # Simple color assignment
  keyword = palette.pine
  string  = palette.gold

  # Color with text styling
  comment {
    color  = palette.muted
    italic = true
  }

  bold {
    color = palette.gold
    bold  = true
  }

  # Nested scopes
  markup {
    heading = palette.love
    link {
      color     = palette.foam
      underline = true
    }
  }
}
```

Style properties (`bold`, `italic`, `underline`) are optional and default to false.

## Templates

Templates transform your theme data into application-specific config files. They live in the `templates/` directory and use Go's text/template syntax with these data structures:

- `.Meta` - name, author, appearance
- `.Palette` - color definitions as a nested tree (values are Style objects)
- `.Theme` - UI color mappings
- `.Syntax` - syntax highlighting rules with optional styles
- `.ANSI` - terminal colors

### Template Functions

**Color Formatting Functions:**
- `hex "path"` - hex with hash prefix (e.g., `#191724`)
- `bhex "path"` - bare hex without hash (e.g., `191724`)
- `hexa "path"` - hex with alpha channel (e.g., `#191724ff`)
- `bhexa "path"` - bare hex with alpha (e.g., `191724ff`)
- `rgb "path"` - RGB function format (e.g., `rgb(25, 23, 36)`)
- `rgba "path"` - RGBA with alpha (e.g., `rgba(25, 23, 36, 1.0)`)

All functions accept dot-notation palette paths (e.g., `"highlight.low"`, `"base"`).

**Style Access:**
- `style "path"` - returns a Style object with `.Bold`, `.Italic`, `.Underline` flags

### Example Templates

**Ghostty terminal** (`ghostty.tmpl`):

```
background = {{ bhex .Theme.background }}
foreground = {{ bhex .Theme.foreground }}
cursor-color = {{ bhex .Theme.cursor }}
```

**Zed editor** (`zed.json.tmpl`):

```json
{
  "name": "{{ .Meta.Name }}",
  "style": {
    "background": "{{ hex .Theme.background }}",
    "editor.background": "{{ hex .Theme.background }}"
  }
}
```

Templates can conditionally output style properties using `if` statements. Each target application receives exactly the config format it expects, all generated from your single source of truth.

## Usage

```bash
# Generate all themes
paletteswap generate

# Generate for specific apps only
paletteswap generate --app ghostty --app zed

# Custom paths
paletteswap generate --theme mytheme.hcl --templates ./templates --out ./themes
```

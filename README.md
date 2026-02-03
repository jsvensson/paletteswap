# PaletteSwap

PaletteSwap generates application-specific color themes from a single theme source file. Define your colors once, then transform them into config files for any terminal, editor, or tool that supports custom themes written in plain text formats.

## HCL Theme Format

**Observe:** This is still in early development. The format is subject to breaking changes.

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

Define your color constants as hex values. The names can be arbitrary.

```hcl
palette {
  base    = "#191724"
  surface = "#1f1d2e"
  text    = "#e0def4"
  love    = "#eb6f92"
  gold    = "#f6c177"
}
```

Palette colors can be referenced by other blocks using `palette.<name>` syntax.

### Theme Block

Maps palette colors to UI elements:

```hcl
theme {
  background = palette.base
  foreground = palette.text
  cursor     = palette.highlight_high
  selection  = palette.highlight_med
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

**Observe:** The syntax names can currently be entirely arbitrary; there is no fixed standard yet. This should be decided upon before PaletteSwap can reliably be adopted. See [issue #5](https://github.com/jsvensson/paletteswap/issues/5).

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
- `.Palette` - raw color definitions
- `.Theme` - UI color mappings
- `.Syntax` - syntax highlighting rules with optional styles
- `.ANSI` - terminal colors

### Template Functions

- `hex` - outputs color as quoted hex (e.g., `"#191724"`)
- `hexBare` - outputs color as bare hex (e.g., `#191724`)

### Example Templates

**Ghostty terminal** (`ghostty.tmpl`):

```
background = {{ hexBare .Theme.background }}
foreground = {{ hexBare .Theme.foreground }}
cursor-color = {{ hexBare .Theme.cursor }}
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

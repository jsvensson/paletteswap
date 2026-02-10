# PaletteSwap

PaletteSwap generates application-specific color themes from a single theme source file. Define your colors once, then transform them into config files for any terminal, editor, or tool that supports custom themes written in plain text formats.

## HCL Theme Format

> [!WARNING]
> PaletteSwap is still in early development. The theme and templates formats are subject to breaking changes.
>
> **BREAKING CHANGES (2026-02-05):**
> - Template functions redesigned: `hexBare` → `bhex`, removed `palette` function
> - Universal path notation: use `hex "block.path"` instead of `palette "path" | hex`
> - Style function requires block prefix: `style "palette.custom"` instead of `style "custom"`
> - ANSI block now required with all 16 colors

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

All palette values are accessible in templates using universal dot-notation paths:

```text
{{ hex "palette.highlight.low" }}
{{ bhex "palette.base" }}
```

To access style flags (bold, italic, underline), use the `style` function with block prefix:

```text
{{ if (style "palette.custom.bold").Bold }}bold{{ end }}
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

**Color formatting functions** accept universal dot-notation paths like `"palette.base"`, `"theme.background"`, `"ansi.black"`, or `"syntax.keyword"`:

- `hex "path"` - hex with hash prefix (e.g., `#191724`)
- `bhex "path"` - bare hex without hash (e.g., `191724`)
- `hexa "path"` - hex with alpha channel (e.g., `#191724ff`)
- `bhexa "path"` - bare hex with alpha (e.g., `191724ff`)
- `rgb "path"` - RGB function format (e.g., `rgb(25, 23, 36)`)
- `rgba "path"` - RGBA with alpha (e.g., `rgba(25, 23, 36, 1.0)`)

**Style access:**

- `style "path"` - returns a Style object with `.Bold`, `.Italic`, `.Underline` flags (supports `palette.*` and `syntax.*` blocks)

### Example Templates

**Ghostty terminal** (`ghostty.tmpl`):

```
background = {{ bhex "theme.background" }}
foreground = {{ bhex "theme.foreground" }}
cursor-color = {{ bhex "theme.cursor" }}
```

**Zed editor** (`zed.json.tmpl`):

```json
{
  "name": "{{ .Meta.Name }}",
  "style": {
    "background": "{{ hex "theme.background" }}",
    "editor.background": "{{ hex "theme.background" }}"
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

## Releases

PaletteSwap uses [semantic versioning](https://semver.org/) and [conventional commits](https://www.conventionalcommits.org/).

### Creating a Release

1. Ensure PR titles follow conventional commit format:
   - `feat: add new feature` → minor version bump
   - `fix: correct bug` → patch version bump
   - `feat!: breaking change` or `BREAKING CHANGE:` in body → major version bump

2. Go to **Actions** → **Create Release PR** → **Run workflow**

3. Review the generated PR which includes:
   - Updated `CHANGELOG.md`
   - Version bump

4. Merge the PR to automatically:
   - Create a git tag
   - Build cross-platform binaries
   - Publish GitHub release with binaries

### Installing from Release

Download the appropriate binary for your platform from the [releases page](https://github.com/jsvensson/paletteswap/releases):

```bash
# macOS (Apple Silicon)
curl -L -o paletteswap.tar.gz https://github.com/jsvensson/paletteswap/releases/latest/download/paletteswap_Darwin_arm64.tar.gz
tar -xzf paletteswap.tar.gz
mv paletteswap /usr/local/bin/

# Linux (x86_64)
curl -L -o paletteswap.tar.gz https://github.com/jsvensson/paletteswap/releases/latest/download/paletteswap_Linux_x86_64.tar.gz
tar -xzf paletteswap.tar.gz
sudo mv paletteswap /usr/local/bin/
```

# Inspiration

With ❤️ to [Rosé Pine](https://rosepinetheme.com/) and [Biscuit](https://github.com/Biscuit-Theme/biscuit), two great color themes.

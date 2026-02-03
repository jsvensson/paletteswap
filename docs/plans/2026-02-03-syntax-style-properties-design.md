# Syntax Style Properties

Add support for font style properties (bold, italic, underline) on syntax scope entries, so theme authors can define both color and style in one place. Templates decide how to apply the style information for each target application.

## HCL Schema

Syntax entries support two forms:

**Simple assignment** (color only, existing behavior):

```hcl
syntax {
  keyword = palette.pine
  string  = palette.gold
}
```

**Block form** (color + style):

```hcl
syntax {
  keyword {
    color     = palette.pine
    bold      = true
  }
  comment {
    color     = palette.muted
    italic    = true
  }
  link {
    color     = palette.foam
    underline = true
  }
}
```

Both forms can be mixed freely within the same syntax block. Nested scopes support the same two forms:

```hcl
syntax {
  markup {
    heading = palette.love
    bold {
      color = palette.gold
      bold  = true
    }
  }
}
```

In block form, `color` is required. `bold`, `italic`, and `underline` are optional booleans that default to `false`.

## Data Model

A new `SyntaxStyle` struct replaces `color.Color` in the syntax map:

```go
type SyntaxStyle struct {
    Color     color.Color
    Bold      bool
    Italic    bool
    Underline bool
}
```

The `Theme` struct's `Syntax` field changes from `map[string]color.Color` to `map[string]SyntaxStyle`. Simple color assignments produce a `SyntaxStyle` with only `Color` set.

## HCL Parsing

The syntax parser handles two cases for each entry in a syntax block:

1. **Attribute** (`keyword = palette.pine`): Parse as color, wrap in `SyntaxStyle{Color: color}` with style bools all `false`.
2. **Block** (`keyword { color = ...; bold = true }`): Parse `color` as a required color attribute. Parse `bold`, `italic`, `underline` as optional bool attributes defaulting to `false`.

Nested block handling (e.g. `syntax.markup.heading`) uses the same two-form logic at every nesting level.

The parser distinguishes attributes from blocks by inspecting the HCL body content and checking each item's type.

## Template Access

Templates access style fields with dot notation:

```
{{ .Syntax.keyword.Color | hex }}
{{ .Syntax.keyword.Bold }}
{{ .Syntax.keyword.Italic }}
{{ .Syntax.keyword.Underline }}
```

The `hex`, `hexBare`, and `rgb` template functions work on the `.Color` field.

Example Zed template syntax entry:

```
"keyword": {
  "color": "{{ .Syntax.keyword.Color | hex }}"
  {{- if .Syntax.keyword.Bold }}, "font_weight": 700{{ end }}
  {{- if .Syntax.keyword.Italic }}, "font_style": "italic"{{ end }}
}
```

## Backward Compatibility

This is a breaking change to the data model. Practical impact is minimal since the project is pre-1.0 with two controlled templates.

- **HCL files**: No changes required. Existing color-only assignments work as-is.
- **Templates**: Must update syntax access from `{{ hex (index .Syntax "keyword") }}` to `{{ .Syntax.keyword.Color | hex }}`.
- **Ghostty template**: Unaffected (uses palette/theme/ANSI only, no syntax entries).

As part of this change, all templates are cleaned up to use dot notation instead of `index` for syntax map access.

## Testing

Config parser test cases:

- Simple assignment parses to `SyntaxStyle` with only `Color` set, style bools `false`
- Block with all styles (`bold`, `italic`, `underline`) parses correctly
- Block with partial styles: omitted styles default to `false`
- Block missing `color` produces a parse error
- Mixed entries (simple assignments and style blocks) in the same syntax block
- Nested blocks with styles (`markup { bold { color = ...; bold = true } }`)
- Existing color tests updated for the `SyntaxStyle` wrapper

Engine/template tests updated to use `.Color` on syntax values.

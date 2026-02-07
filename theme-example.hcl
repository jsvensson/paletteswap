# The meta block contains metadata about the theme.
meta {
  name       = "Example theme"
  author     = "Bob Loblaw"
  appearance = "dark"
  url        = "https://example.com"
}

# Palette defines reusable color references.
palette {
  black = "#000000"
  white = "#ffffff"
  gray  = "#c0c0c0"

  # Palettes can also contain nested palettes.
  highlight {
    low  = "#21202e"
    mid  = "#403d52"
    high = "#524f67"
  }
}


# Theme contains elements for the user interface.
theme {
  # Reference colors from the palette.
  background = palette.black
  foreground = palette.white
}

# syntax contains elements for syntax highlighting.
syntax {
  function = palette.gray

  # Blocks can be used for enhanced styling, such as bold/italic.
  comment {
    # `style` is a reserved keyword for block styling.
    style {
      # Colors can be modified using functions.
      color  = darken(palette.white, 0.2)
      italic = true # and/or bold, underline
    }

    # Blocks can contain further nested syntax definitions.
    # This example can be accessed through `syntax.comment.doc`.
    doc = palette.gray
  }
}

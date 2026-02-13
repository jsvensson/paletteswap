package format

import (
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jsvensson/paletteswap/internal/theme"
)

var multipleBlankLines = regexp.MustCompile(`\n{3,}`)
var blankLineAfterOpenBrace = regexp.MustCompile(`\{\n\s*\n`)
var blankLineBeforeCloseBrace = regexp.MustCompile(`\n\s*\n(\s*\})`)

// Format takes HCL source content and returns it formatted according to
// HCL canonical style rules. It uses hclwrite.Format which handles
// indentation, spacing, and newline normalization.
//
// The formatter works even on partial/invalid HCL, making it suitable
// for use while the user is still typing.
func Format(content string) (string, error) {
	formatted := hclwrite.Format([]byte(content))
	// Reorder ANSI block attributes to canonical order.
	formatted = reorderANSIBlock(formatted)
	// Collapse multiple consecutive blank lines into a single blank line.
	collapsed := multipleBlankLines.ReplaceAllString(string(formatted), "\n\n")
	// Remove blank lines immediately after opening braces.
	collapsed = blankLineAfterOpenBrace.ReplaceAllString(collapsed, "{\n")
	// Remove blank lines immediately before closing braces.
	collapsed = blankLineBeforeCloseBrace.ReplaceAllString(collapsed, "\n${1}")
	return collapsed, nil
}

// ansiBlockPattern matches the "ansi {" opening and captures everything
// between the opening brace and the closing brace.
var ansiBlockPattern = regexp.MustCompile(`(?s)(ansi\s*\{)\n(.*?)\n(\})`)

// reorderANSIBlock finds the "ansi" block in the formatted HCL source and
// reorders its attributes to match the canonical order defined in
// theme.RequiredANSIColors. Comments and blank lines immediately preceding
// an attribute travel with that attribute.
func reorderANSIBlock(src []byte) []byte {
	// First verify this is valid HCL with an ansi block using the parser.
	file, diags := hclwrite.ParseConfig(src, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return src
	}

	hasANSI := false
	for _, block := range file.Body().Blocks() {
		if block.Type() == "ansi" {
			hasANSI = true
			break
		}
	}
	if !hasANSI {
		return src
	}

	// Use regex to find and replace the inner content of the ansi block.
	return ansiBlockPattern.ReplaceAllFunc(src, func(match []byte) []byte {
		loc := ansiBlockPattern.FindSubmatchIndex(match)
		// loc[2]:loc[3] = "ansi {"
		// loc[4]:loc[5] = inner content
		// loc[6]:loc[7] = "}"
		opener := match[loc[2]:loc[3]]
		inner := string(match[loc[4]:loc[5]])
		closer := match[loc[6]:loc[7]]

		lines := strings.Split(inner, "\n")
		reordered := reorderEntries(lines, theme.RequiredANSIColors)
		newInner := strings.Join(reordered, "\n")

		var result []byte
		result = append(result, opener...)
		result = append(result, '\n')
		result = append(result, []byte(newInner)...)
		result = append(result, '\n')
		result = append(result, closer...)
		return result
	})
}

// entry represents a single attribute and any comment/blank lines that precede it.
type entry struct {
	name  string   // attribute name (empty for trailing non-attribute lines)
	lines []string // all lines belonging to this entry (comments + attribute)
}

// reorderEntries takes lines from inside an ANSI block and reorders the
// attribute entries according to the given canonical order. Comment and blank
// lines immediately before an attribute are grouped with that attribute.
// Unknown attributes are appended at the end. After reordering, attribute
// lines are realigned so that all '=' signs are at the same column.
func reorderEntries(lines []string, order []string) []string {
	var entries []entry
	var pending []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			pending = append(pending, line)
			continue
		}
		if eqIdx := strings.Index(trimmed, "="); eqIdx != -1 {
			// This is an attribute line. Group it with any pending comment/blank lines.
			attrLines := make([]string, 0, len(pending)+1)
			attrLines = append(attrLines, pending...)
			attrLines = append(attrLines, line)
			pending = nil

			// Extract attribute name: text before '=' trimmed of whitespace.
			name := strings.TrimSpace(trimmed[:eqIdx])
			entries = append(entries, entry{name: name, lines: attrLines})
		} else {
			pending = append(pending, line)
		}
	}

	// Build an index map from the canonical order.
	orderIndex := make(map[string]int, len(order))
	for i, name := range order {
		orderIndex[name] = i
	}

	// Map entries by attribute name for lookup.
	entryByName := make(map[string]entry, len(entries))
	var unknownEntries []entry
	for _, e := range entries {
		if _, ok := orderIndex[e.name]; ok {
			entryByName[e.name] = e
		} else {
			unknownEntries = append(unknownEntries, e)
		}
	}

	// Emit entries in canonical order.
	var result []string
	for _, name := range order {
		if e, ok := entryByName[name]; ok {
			result = append(result, e.lines...)
		}
	}

	// Append any unknown entries at the end.
	for _, e := range unknownEntries {
		result = append(result, e.lines...)
	}

	// Append any trailing pending lines (comments/blanks after last attribute).
	result = append(result, pending...)

	// Realign all attribute lines so '=' signs are at the same column.
	result = alignAttributes(result)

	return result
}

// alignAttributes normalizes the alignment of attribute lines so that all '='
// signs line up at the same column. Non-attribute lines (comments, blanks) are
// left unchanged. The indentation prefix (leading whitespace) is preserved.
func alignAttributes(lines []string) []string {
	// First pass: find the longest attribute name across all attribute lines.
	maxNameLen := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			continue
		}
		name := strings.TrimRight(trimmed[:eqIdx], " ")
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	if maxNameLen == 0 {
		return lines
	}

	// Second pass: rewrite attribute lines with uniform padding.
	result := make([]string, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			result[i] = line
			continue
		}
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			result[i] = line
			continue
		}

		// Determine the indentation prefix.
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		name := strings.TrimRight(trimmed[:eqIdx], " ")
		rest := trimmed[eqIdx:] // "= value" or "= value # comment"
		padding := strings.Repeat(" ", maxNameLen-len(name))
		result[i] = indent + name + padding + " " + rest
	}

	return result
}

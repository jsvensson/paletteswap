package format

import (
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

var multipleBlankLines = regexp.MustCompile(`\n{3,}`)

// Format takes HCL source content and returns it formatted according to
// HCL canonical style rules. It uses hclwrite.Format which handles
// indentation, spacing, and newline normalization.
//
// The formatter works even on partial/invalid HCL, making it suitable
// for use while the user is still typing.
func Format(content string) (string, error) {
	formatted := hclwrite.Format([]byte(content))
	// Collapse multiple consecutive blank lines into a single blank line.
	collapsed := multipleBlankLines.ReplaceAllString(string(formatted), "\n\n")
	return collapsed, nil
}

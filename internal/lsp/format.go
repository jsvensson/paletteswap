package lsp

import (
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// format takes HCL source content and returns it formatted according to
// HCL canonical style rules. It uses hclwrite.Format which handles
// indentation, spacing, and newline normalization.
//
// The formatter works even on partial/invalid HCL, making it suitable
// for use while the user is still typing.
func format(content string) (string, error) {
	formatted := hclwrite.Format([]byte(content))
	return string(formatted), nil
}

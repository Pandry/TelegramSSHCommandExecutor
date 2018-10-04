package utils

import "strings"

//RemoveMarkdownSyntax removes the MD syntax (* ,` and _)
func RemoveMarkdownSyntax(s string) string {
	return strings.Replace(strings.Replace(strings.Replace(s, "_", "", -1), "*", "", -1), "`", "", -1)
}

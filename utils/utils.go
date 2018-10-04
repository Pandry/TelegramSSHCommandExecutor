package utils

import "strings"

//RemoveMarkdownSyntax removes the MD syntax (* ,` and _)
func RemoveMarkdownSyntax(s string) string {
	return strings.Replace(strings.Replace(strings.Replace(s, "_", "", -1), "*", "", -1), "`", "", -1)
}

//EscapeXMLTags escapes the HTML tags (* ,` and _)
func EscapeXMLTags(s string) string {
	return strings.Replace(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					s, "\"", "&quot;", -1),
				"&", "&amp;", -1),
			">", "&gt;", -1),
		"<", "&lt;", -1)
}

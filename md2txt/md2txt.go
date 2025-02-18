package md2txt

import (
	"html"
	"regexp"

	"github.com/russross/blackfriday/v2"
)

func Markdown2Text(markdown []byte) string {
	// Step 1: Convert Markdown to HTML
	htmlContent := blackfriday.Run(markdown)

	// Step 2: Strip HTML tags
	plainText := stripHTML(htmlContent)

	return plainText
}

// stripHTML removes HTML tags from the content
func stripHTML(content []byte) string {
	// Using a regular expression to remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(string(content), "")

	// Trim spaces and unescape any HTML entities
	cleaned = html.UnescapeString(cleaned)
	cleaned = regexp.MustCompile(`[ \t\r\f]`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\n{5,}`).ReplaceAllString(cleaned, "\n\n\n")

	return cleaned
}

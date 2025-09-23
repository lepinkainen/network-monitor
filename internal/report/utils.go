package report

import "strings"

// sanitizeFilename replaces dots and special characters for safe filenames
func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		".", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		" ", "_",
	)
	return replacer.Replace(s)
}

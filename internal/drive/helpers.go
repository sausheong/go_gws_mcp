// Package drive implements Google Drive MCP tools.
package drive

import "strings"

// MIME constants for Google native types.
const (
	MimeTypeGoogleDoc    = "application/vnd.google-apps.document"
	MimeTypeGoogleSheet  = "application/vnd.google-apps.spreadsheet"
	MimeTypeGoogleSlides = "application/vnd.google-apps.presentation"
	MimeTypeGoogleForm   = "application/vnd.google-apps.form"
	MimeTypeFolder       = "application/vnd.google-apps.folder"
)

// MaxContentBytes caps inline file content in get_drive_file_content responses.
const MaxContentBytes = 5 * 1024 * 1024 // 5 MiB

// exportMimeFor returns the export MIME for a Google native type, or ""
// if the file should be downloaded as-is.
func exportMimeFor(mt string) string {
	switch mt {
	case MimeTypeGoogleDoc:
		return "text/plain"
	case MimeTypeGoogleSheet:
		return "text/csv"
	case MimeTypeGoogleSlides:
		return "text/plain"
	}
	return ""
}

// escapeDriveString escapes a value for inclusion as a single-quoted literal
// in a Drive query string (https://developers.google.com/drive/api/guides/search-files#query_string_terms_and_operators).
// Backslash and apostrophe are the only characters Drive's query syntax requires
// to be escaped inside a single-quoted literal.
func escapeDriveString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// isTextLike reports whether a non-Google MIME is plain enough to inline as text.
func isTextLike(mt string) bool {
	if strings.HasPrefix(mt, "text/") {
		return true
	}
	switch mt {
	case "application/json", "application/xml",
		"application/javascript", "application/x-yaml":
		return true
	}
	return false
}

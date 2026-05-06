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

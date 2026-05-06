// Package gmail implements Gmail MCP tools.
package gmail

import (
	"encoding/base64"
	"html"
	"regexp"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// ExtractTextBody walks a Gmail MessagePart tree and returns the best
// available textual body. Prefers text/plain, falls back to a stripped
// version of text/html.
func ExtractTextBody(payload *gmailapi.MessagePart) string {
	if payload == nil {
		return ""
	}
	plain, htmlBody := walkParts(payload)
	if plain != "" {
		return strings.TrimSpace(plain)
	}
	if htmlBody != "" {
		return stripHTML(htmlBody)
	}
	return ""
}

func walkParts(part *gmailapi.MessagePart) (plain, htmlBody string) {
	if part == nil {
		return "", ""
	}
	if part.Body != nil && part.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			switch part.MimeType {
			case "text/plain":
				if plain == "" {
					plain = string(decoded)
				}
			case "text/html":
				if htmlBody == "" {
					htmlBody = string(decoded)
				}
			}
		}
	}
	for _, p := range part.Parts {
		subPlain, subHTML := walkParts(p)
		if plain == "" {
			plain = subPlain
		}
		if htmlBody == "" {
			htmlBody = subHTML
		}
	}
	return plain, htmlBody
}

var tagRE = regexp.MustCompile(`<[^>]+>`)
var spaceRE = regexp.MustCompile(`\s+`)

func stripHTML(in string) string {
	out := tagRE.ReplaceAllString(in, " ")
	out = html.UnescapeString(out)
	return strings.TrimSpace(spaceRE.ReplaceAllString(out, " "))
}

// HeaderValue returns the value of a header (case-insensitive lookup).
func HeaderValue(headers []*gmailapi.MessagePartHeader, name string) string {
	target := strings.ToLower(name)
	for _, h := range headers {
		if strings.ToLower(h.Name) == target {
			return h.Value
		}
	}
	return ""
}

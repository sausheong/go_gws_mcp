// Package apierror wraps Google API errors with LLM-targeted messages that
// embed actionable recovery steps.
package apierror

import (
	"errors"
	"fmt"

	"google.golang.org/api/googleapi"
)

// Format returns a string suitable for an MCP tool result body. Mirrors
// Python's @handle_http_errors decorator behavior.
func Format(toolName string, err error) string {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		switch gerr.Code {
		case 403:
			if isAccessNotConfigured(gerr) {
				return apiEnablementMessage(toolName, gerr)
			}
			return authErrorMessage(toolName, gerr)
		case 401:
			return authErrorMessage(toolName, gerr)
		default:
			return fmt.Sprintf("API error in %s: %s (HTTP %d)", toolName, gerr.Message, gerr.Code)
		}
	}
	return fmt.Sprintf("Unexpected error in %s: %v", toolName, err)
}

func isAccessNotConfigured(g *googleapi.Error) bool {
	for _, item := range g.Errors {
		if item.Reason == "accessNotConfigured" {
			return true
		}
	}
	return false
}

func apiEnablementMessage(toolName string, g *googleapi.Error) string {
	return fmt.Sprintf(
		`API error in %s: The required Google API is not enabled for your project.

Details: %s

Enable the API in the Google Cloud Console (search for the relevant API by name and click 'Enable').
After enabling, wait 1-2 minutes for the change to propagate, then retry.

IMPORTANT - LLM: instruct the user to enable the required API and surface the Cloud Console link as a clickable hyperlink.`,
		toolName, g.Message,
	)
}

func authErrorMessage(toolName string, g *googleapi.Error) string {
	return fmt.Sprintf(
		`API error in %s: %s (HTTP %d).
You might need to re-authenticate.
LLM: ask the user to re-run start_google_auth (or re-authenticate via their MCP client) and then retry the original command.`,
		toolName, g.Message, g.Code,
	)
}

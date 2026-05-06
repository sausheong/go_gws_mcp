package sheets

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// CreateArgs is the arg shape for create_spreadsheet.
type CreateArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	Title           string `json:"title"`
}

// CreateSpreadsheet creates a new Google Sheets spreadsheet with the given title.
// The new spreadsheet contains a default Sheet1.
func CreateSpreadsheet(ctx context.Context, svc *sheetsapi.Service, userEmail string, a CreateArgs) (string, error) {
	if a.Title == "" {
		return "", errors.New("title is required")
	}
	created, err := svc.Spreadsheets.Create(&sheetsapi.Spreadsheet{
		Properties: &sheetsapi.SpreadsheetProperties{Title: a.Title},
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created spreadsheet %q (id: %s) — %s",
		created.Properties.Title, created.SpreadsheetId, created.SpreadsheetUrl), nil
}

func registerCreate(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("create_spreadsheet",
		mcp.WithDescription("Creates a new Google Sheets spreadsheet with the given title (and a default Sheet1)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Spreadsheet title")),
	)
	scopes := []string{auth.SheetsScope}
	reg.Record("create_spreadsheet", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("create_spreadsheet", "Sheets", scopes, sheetsFactory, c, email, CreateSpreadsheet))
}

package sheets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// ReadArgs is the arg shape for read_sheet_values.
type ReadArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	SpreadsheetID   string `json:"spreadsheet_id"`
	Range           string `json:"range"` // A1 notation, e.g. "Sheet1!A1:D10"
}

// ReadSheetValues reads cell values from an A1-notation range and renders them
// as one row per line, cells joined by " | ".
func ReadSheetValues(ctx context.Context, svc *sheetsapi.Service, userEmail string, a ReadArgs) (string, error) {
	if a.SpreadsheetID == "" {
		return "", errors.New("spreadsheet_id is required")
	}
	if a.Range == "" {
		return "", errors.New("range is required (e.g., \"Sheet1!A1:D10\")")
	}

	resp, err := svc.Spreadsheets.Values.Get(a.SpreadsheetID, a.Range).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Spreadsheet: %s\nRange: %s\nRows: %d\n---\n",
		a.SpreadsheetID, resp.Range, len(resp.Values))
	for _, row := range resp.Values {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = fmt.Sprintf("%v", v)
		}
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString("\n")
	}
	return b.String(), nil
}

func registerRead(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("read_sheet_values",
		mcp.WithDescription("Reads cell values from a sheet range in A1 notation (e.g., \"Sheet1!A1:D10\"). Empty cells are returned as empty strings."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("Spreadsheet ID")),
		mcp.WithString("range", mcp.Required(), mcp.Description("A1-notation range, e.g. 'Sheet1!A1:D10'")),
	)
	scopes := []string{auth.SheetsReadonlyScope}
	reg.Record("read_sheet_values", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("read_sheet_values", "Sheets", scopes, sheetsFactory, c, email, ReadSheetValues))
}

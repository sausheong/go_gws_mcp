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

// InfoArgs is the arg shape for get_spreadsheet_info.
type InfoArgs struct {
	UserGoogleEmail string `json:"user_google_email"`
	SpreadsheetID   string `json:"spreadsheet_id"`
}

// GetSpreadsheetInfo returns the spreadsheet title + sheet inventory (name + grid size).
// Uses Spreadsheets.Get with no ranges — pulls only metadata.
func GetSpreadsheetInfo(ctx context.Context, svc *sheetsapi.Service, userEmail string, a InfoArgs) (string, error) {
	if a.SpreadsheetID == "" {
		return "", errors.New("spreadsheet_id is required")
	}
	ss, err := svc.Spreadsheets.Get(a.SpreadsheetID).
		Fields("spreadsheetId, properties.title, sheets.properties(sheetId,title,index,gridProperties)").
		Context(ctx).Do()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	title := ""
	if ss.Properties != nil {
		title = ss.Properties.Title
	}
	fmt.Fprintf(&b, "Spreadsheet: %s\nID: %s\nSheets: %d\n---\n", title, ss.SpreadsheetId, len(ss.Sheets))
	for _, sh := range ss.Sheets {
		if sh.Properties == nil {
			continue
		}
		rows, cols := int64(0), int64(0)
		if sh.Properties.GridProperties != nil {
			rows = sh.Properties.GridProperties.RowCount
			cols = sh.Properties.GridProperties.ColumnCount
		}
		fmt.Fprintf(&b, "- [%d] %s (id: %d, %dx%d)\n",
			sh.Properties.Index, sh.Properties.Title, sh.Properties.SheetId, rows, cols)
	}
	return b.String(), nil
}

func registerInfo(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("get_spreadsheet_info",
		mcp.WithDescription("Returns spreadsheet title + per-sheet metadata (name, index, grid size). Does not return cell values; use read_sheet_values for that."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("Google Sheets spreadsheet ID")),
	)
	scopes := []string{auth.SheetsReadonlyScope}
	reg.Record("get_spreadsheet_info", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("get_spreadsheet_info", "Sheets", scopes, sheetsFactory, c, email, GetSpreadsheetInfo))
}

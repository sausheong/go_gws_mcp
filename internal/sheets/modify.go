package sheets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	sheetsapi "google.golang.org/api/sheets/v4"

	"github.com/sausheong/go_gws_mcp/internal/auth"
	"github.com/sausheong/go_gws_mcp/internal/core"
)

// ModifyArgs is the arg shape for modify_sheet_values.
type ModifyArgs struct {
	UserGoogleEmail  string          `json:"user_google_email"`
	SpreadsheetID    string          `json:"spreadsheet_id"`
	Range            string          `json:"range"`                        // A1 notation
	Values           json.RawMessage `json:"values"`                       // 2D array of cell values
	ValueInputOption string          `json:"value_input_option,omitempty"` // "USER_ENTERED" (default) or "RAW"
}

// ModifySheetValues writes a 2D array of values to an A1 range.
// `values` is a JSON 2D array of arbitrary scalars (strings, numbers, bools).
func ModifySheetValues(ctx context.Context, svc *sheetsapi.Service, userEmail string, a ModifyArgs) (string, error) {
	if a.SpreadsheetID == "" {
		return "", errors.New("spreadsheet_id is required")
	}
	if a.Range == "" {
		return "", errors.New("range is required")
	}
	if len(a.Values) == 0 {
		return "", errors.New("values is required (2D array)")
	}

	var rows [][]interface{}
	if err := json.Unmarshal(a.Values, &rows); err != nil {
		return "", fmt.Errorf("values must be a JSON 2D array: %w", err)
	}

	opt := a.ValueInputOption
	if opt == "" {
		opt = "USER_ENTERED"
	}
	if opt != "USER_ENTERED" && opt != "RAW" {
		return "", fmt.Errorf("value_input_option must be USER_ENTERED or RAW (got %q)", opt)
	}

	body := &sheetsapi.ValueRange{
		Range:  a.Range,
		Values: rows,
	}

	resp, err := svc.Spreadsheets.Values.Update(a.SpreadsheetID, a.Range, body).
		ValueInputOption(opt).
		Context(ctx).Do()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Updated %s — %d cells across %d row(s) (mode: %s)",
		resp.UpdatedRange, resp.UpdatedCells, resp.UpdatedRows, opt), nil
}

func registerModify(srv *server.MCPServer, reg *core.Registry, c *auth.OAuthClient, email string) {
	tool := mcp.NewTool("modify_sheet_values",
		mcp.WithDescription("Writes a 2D array of values to a sheet range. value_input_option controls formula parsing: USER_ENTERED (default; parses formulas, dates) or RAW (writes values verbatim)."),
		mcp.WithString("user_google_email", mcp.Required(), mcp.Description("User's Google email address")),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("Spreadsheet ID")),
		mcp.WithString("range", mcp.Required(), mcp.Description("A1-notation range, e.g. 'Sheet1!A1:D10'")),
		mcp.WithArray("values",
			mcp.Required(),
			mcp.Description("2D array of cell values (rows of arrays of scalars)"),
			mcp.Items(map[string]any{"type": "array", "items": map[string]any{}}),
		),
		mcp.WithString("value_input_option", mcp.Description("USER_ENTERED (default) or RAW")),
	)
	scopes := []string{auth.SheetsScope}
	reg.Record("modify_sheet_values", scopes)
	srv.AddTool(tool, auth.RequireGoogleService("modify_sheet_values", "Sheets", scopes, sheetsFactory, c, email, ModifySheetValues))
}

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkjdaniel/gogeek/v2"
	bggrequest "github.com/kkjdaniel/gogeek/v2/request"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// geekListBaseURL is on the legacy XML API1, not xmlapi2 — GeekLists were
// never ported to API2, and gogeek has no package for either. Still reuses
// bggrequest.FetchAndUnmarshal (URL-agnostic auth/rate-limit/retry/XML-cleanup
// logic) rather than hand-rolling a new HTTP call.
const geekListBaseURL = "https://boardgamegeek.com/xmlapi/geeklist/"

type geekList struct {
	Title    string         `xml:"title"`
	Username string         `xml:"username"`
	PostDate string         `xml:"postdate"`
	EditDate string         `xml:"editdate"`
	Thumbs   int            `xml:"thumbs"`
	NumItems int            `xml:"numitems"`
	Items    []geekListItem `xml:"item"`
}

type geekListItem struct {
	ID         int    `xml:"id,attr"`
	ObjectID   int    `xml:"objectid,attr"`
	ObjectType string `xml:"objecttype,attr"`
	ObjectName string `xml:"objectname,attr"`
	Username   string `xml:"username,attr"`
	PostDate   string `xml:"postdate,attr"`
	Body       string `xml:"body"`
}

func GeeklistTool(client *gogeek.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-geeklist",
		mcp.WithDescription("Read a BGG GeekList (a community-curated, ordered list of games/items with commentary) by id. Uses BGG's legacy XML API1 since GeekLists aren't part of XML API2. Read-only — BGG's API does not support creating or modifying GeekLists at all."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("The GeekList id")),
		mcp.WithBoolean("comments", mcp.Description("Include per-item user comments (larger, slower response). Defaults to false.")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := req.GetArguments()
		idVal, ok := arguments["id"].(float64)
		if !ok {
			return mcp.NewToolResultText("id is required"), nil
		}
		listID := int(idVal)

		requestURL := fmt.Sprintf("%s%d", geekListBaseURL, listID)
		if boolField(arguments, "comments", false) {
			requestURL += "?comments=1"
		}

		var list geekList
		if err := bggrequest.FetchAndUnmarshal(client, requestURL, &list); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error fetching geeklist %d: %v", listID, err)), nil
		}

		out, err := json.Marshal(list)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

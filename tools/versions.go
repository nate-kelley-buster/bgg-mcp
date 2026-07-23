package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kkjdaniel/gogeek/v2"
	"github.com/kkjdaniel/gogeek/v2/constants"
	bggrequest "github.com/kkjdaniel/gogeek/v2/request"
	"github.com/kkjdaniel/gogeek/v2/thing"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// gogeek's thing package has no support for the versions=1 query flag or the
// resulting <versions> XML block, so these types are local to bgg-mcp. This
// reuses gogeek's request.FetchAndUnmarshal directly (it's URL-agnostic —
// the same GET/auth-header/rate-limit/retry/XML-cleanup logic every other
// read tool relies on) rather than hand-rolling a new HTTP call, and reuses
// gogeek's exported Name/IntValue/FloatValue/StringValue/Link types for
// consistency with the rest of the codebase.
type itemWithVersions struct {
	Type     string        `xml:"type,attr"`
	ID       int           `xml:"id,attr"`
	Name     []thing.Name  `xml:"name"`
	Versions []gameVersion `xml:"versions>item"`
}

type itemsWithVersions struct {
	Items []itemWithVersions `xml:"item"`
}

type gameVersion struct {
	ID            int               `xml:"id,attr"`
	Name          []thing.Name      `xml:"name"`
	YearPublished thing.IntValue    `xml:"yearpublished"`
	Thumbnail     string            `xml:"thumbnail"`
	Image         string            `xml:"image"`
	ProductCode   thing.StringValue `xml:"productcode"`
	Width         thing.FloatValue  `xml:"width"`
	Length        thing.FloatValue  `xml:"length"`
	Depth         thing.FloatValue  `xml:"depth"`
	Weight        thing.FloatValue  `xml:"weight"`
	Links         []thing.Link      `xml:"link"`
}

func VersionsTool(client *gogeek.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-versions",
		mcp.WithDescription("Get edition/printing/version data for a game — different publishers, languages, dimensions, product codes. Uses BGG's official read-only XML API2 with the versions flag, same reliability as every other read tool (unlike the write tools, nothing here is unverified)."),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := req.GetArguments()
		gameID, _, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		params := url.Values{}
		params.Set("id", fmt.Sprintf("%d", gameID))
		params.Set("versions", "1")
		requestURL := constants.ThingEndpoint + "?" + params.Encode()

		var items itemsWithVersions
		if err := bggrequest.FetchAndUnmarshal(client, requestURL, &items); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error fetching versions for game %d: %v", gameID, err)), nil
		}
		if len(items.Items) == 0 || len(items.Items[0].Versions) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No versions found for game %d", gameID)), nil
		}

		out, err := json.Marshal(items.Items[0].Versions)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

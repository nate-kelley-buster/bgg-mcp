package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kkjdaniel/gogeek/v2"
	"github.com/kkjdaniel/gogeek/v2/plays"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func PlaysTool(client *gogeek.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-plays",
		mcp.WithDescription("Retrieve a user's play history from BoardGameGeek (BGG). Returns logged plays including game name, date, duration, location, players, and win/loss results. Returns up to 100 most recent plays."),
		mcp.WithString("username",
			mcp.Required(),
			mcp.Description("The BGG username to fetch play history for. When the user refers to themselves (me, my, I), use 'SELF' as the value."),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		username, ok := arguments["username"].(string)
		if !ok || username == "" {
			return mcp.NewToolResultText("username is required"), nil
		}

		if username == "SELF" {
			envUsername := os.Getenv("BGG_USERNAME")
			if envUsername == "" {
				return mcp.NewToolResultText("BGG_USERNAME environment variable not set. Either set it or provide your specific username instead of 'SELF'."), nil
			}
			username = envUsername
		}

		result, err := plays.Query(client, username)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error fetching plays for %s: %v", username, err)), nil
		}

		if len(result.Plays) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No plays found for user %s", username)), nil
		}

		out, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

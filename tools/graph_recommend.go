package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func GraphRecommendTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgc-graph-recommend",
		mcp.WithDescription("Get board game recommendations using BGCollector's graph database. Traverses the Neo4j knowledge graph to find games related by shared mechanics, categories, designers, publishers, or player communities. More contextually aware than bgg-recommender (which uses collaborative filtering only). Requires BGCollector backend."),
		mcp.WithNumber("game_id",
			mcp.Description("BGG ID of the seed game to base recommendations on."),
		),
		mcp.WithString("game_name",
			mcp.Description("Name of the seed game. Used when game_id is not known."),
		),
		mcp.WithString("username",
			mcp.Description("BGG username to personalize recommendations based on their collection and play history. Use 'SELF' to reference the configured user."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of recommendations to return (default: 10)"),
		),
		mcp.WithString("strategy",
			mcp.Description("Graph traversal strategy (default: 'balanced')"),
			mcp.Enum("mechanic", "category", "designer", "community", "balanced"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		params := url.Values{}

		if gameID, ok := arguments["game_id"].(float64); ok && gameID > 0 {
			params.Set("game_id", fmt.Sprintf("%d", int(gameID)))
		} else if gameName, ok := arguments["game_name"].(string); ok && gameName != "" {
			params.Set("game_name", gameName)
		} else {
			return mcp.NewToolResultText("Either 'game_id' or 'game_name' must be provided"), nil
		}

		if username, ok := arguments["username"].(string); ok && username != "" {
			resolved := username
			if username == "SELF" {
				envUsername := os.Getenv("BGG_USERNAME")
				if envUsername != "" {
					resolved = envUsername
				}
			}
			if resolved != "SELF" {
				params.Set("username", resolved)
			}
		}

		limit := 10
		if l, ok := arguments["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		params.Set("limit", fmt.Sprintf("%d", limit))

		strategy := "balanced"
		if s, ok := arguments["strategy"].(string); ok && s != "" {
			strategy = s
		}
		params.Set("strategy", strategy)

		baseURL := os.Getenv("BGC_API_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}

		resp, err := http.Get(baseURL + "/api/recommend/graph?" + params.Encode())
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("BGCollector graph recommendations unavailable (backend not running?): %v", err)), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return mcp.NewToolResultText(fmt.Sprintf("BGCollector graph API returned status %d — backend may not have this endpoint yet", resp.StatusCode)), nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error reading response: %v", err)), nil
		}

		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error parsing response: %v", err)), nil
		}

		out, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

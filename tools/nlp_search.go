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

func NLPSearchTool() (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgc-nlp-search",
		mcp.WithDescription("Search the BGCollector database using natural language queries powered by semantic embeddings. Unlike bgg-search (keyword matching), this finds games by meaning: e.g. 'games like chess but faster', 'cooperative horror games for families', 'engine builders with low downtime'. Requires BGCollector backend to be running."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language description of the type of game you're looking for. Be descriptive — the more context, the better the results."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 10, max: 50)"),
		),
		mcp.WithNumber("min_similarity",
			mcp.Description("Minimum similarity score threshold between 0 and 1 (default: 0.6)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		query, ok := arguments["query"].(string)
		if !ok || query == "" {
			return mcp.NewToolResultText("query is required"), nil
		}

		limit := 10
		if l, ok := arguments["limit"].(float64); ok && l > 0 {
			limit = int(l)
			if limit > 50 {
				limit = 50
			}
		}

		minSimilarity := 0.6
		if ms, ok := arguments["min_similarity"].(float64); ok && ms > 0 {
			minSimilarity = ms
		}

		baseURL := os.Getenv("BGC_API_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}

		params := url.Values{}
		params.Set("q", query)
		params.Set("limit", fmt.Sprintf("%d", limit))
		params.Set("min_similarity", fmt.Sprintf("%.2f", minSimilarity))

		resp, err := http.Get(baseURL + "/api/search/nlp?" + params.Encode())
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("BGCollector NLP search unavailable (backend not running?): %v", err)), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return mcp.NewToolResultText(fmt.Sprintf("BGCollector NLP search API returned status %d — backend may not have this endpoint yet", resp.StatusCode)), nil
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

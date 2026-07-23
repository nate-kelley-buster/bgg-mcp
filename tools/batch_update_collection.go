package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/faegents/bgg-mcp/session"
	"github.com/kkjdaniel/gogeek/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// maxBatchSize caps bgg-batch-update-collection. Writes are serialized
// through the shared rate limiter (plus a search round-trip per name-based
// item), so a large batch can take long enough to exceed an MCP client's
// tool-call timeout — better to reject upfront with a clear message than to
// silently run long and get aborted mid-way.
const maxBatchSize = 30

type batchResult struct {
	Index   int    `json:"index"`
	Input   string `json:"input"`
	GameID  int    `json:"game_id,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type batchSummary struct {
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
	Results   []batchResult `json:"results"`
}

// statusToCollectionUpdate maps a batch item's status string to the
// corresponding collectionStatusUpdate flag. "remove" is a special case that
// clears every flag, matching RemoveFromCollectionTool's behavior.
func statusToCollectionUpdate(status string, value bool) (collectionStatusUpdate, error) {
	switch status {
	case "own":
		return collectionStatusUpdate{Own: &value}, nil
	case "want":
		return collectionStatusUpdate{Want: &value}, nil
	case "wanttoplay":
		return collectionStatusUpdate{WantToPlay: &value}, nil
	case "wanttobuy":
		return collectionStatusUpdate{WantToBuy: &value}, nil
	case "fortrade":
		return collectionStatusUpdate{ForTrade: &value}, nil
	case "preordered":
		return collectionStatusUpdate{Preordered: &value}, nil
	case "prevowned":
		return collectionStatusUpdate{PrevOwned: &value}, nil
	case "wishlist":
		return collectionStatusUpdate{Wishlist: &value}, nil
	case "remove":
		f := false
		return collectionStatusUpdate{
			Own: &f, PrevOwned: &f, ForTrade: &f, Want: &f,
			WantToPlay: &f, WantToBuy: &f, Wishlist: &f, Preordered: &f,
		}, nil
	default:
		return collectionStatusUpdate{}, fmt.Errorf("unknown status %q (expected one of: own, want, wanttoplay, wanttobuy, fortrade, preordered, prevowned, wishlist, remove)", status)
	}
}

func BatchUpdateCollectionTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-batch-update-collection",
		mcp.WithDescription(fmt.Sprintf("Apply a collection status change to many games in one call — e.g. bulk-marking a list of games as owned. Each item is a name or id plus a status (own, want, wanttoplay, wanttobuy, fortrade, preordered, prevowned, wishlist, remove) and an optional value (defaults to true; pass false to unset a flag). Items are processed independently: one bad name or failed write never aborts the rest — check the per-item results for failures. Serialized through a shared rate limit against BGG, so keep batches under %d items to avoid a slow call.", maxBatchSize)),
		mcp.WithArray("updates", mcp.Required(), mcp.Description("List of items, each: {name or id, status, value (optional, default true)}")),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		rawItems, ok := arguments["updates"].([]interface{})
		if !ok || len(rawItems) == 0 {
			return mcp.NewToolResultText("updates is required and must be a non-empty array"), nil
		}
		if len(rawItems) > maxBatchSize {
			return mcp.NewToolResultText(fmt.Sprintf("Batch of %d exceeds the %d-item limit — split into smaller batches to avoid a slow or timed-out call.", len(rawItems), maxBatchSize)), nil
		}

		results := make([]batchResult, 0, len(rawItems))
		succeeded := 0

		for i, raw := range rawItems {
			item, ok := raw.(map[string]interface{})
			if !ok {
				results = append(results, batchResult{Index: i, Success: false, Error: "item must be an object"})
				continue
			}

			label := stringField(item, "name", "")
			if label == "" {
				if idVal, ok := item["id"].(float64); ok {
					label = fmt.Sprintf("id:%d", int(idVal))
				}
			}

			status := stringField(item, "status", "")
			if status == "" {
				results = append(results, batchResult{Index: i, Input: label, Success: false, Error: "status is required"})
				continue
			}

			update, err := statusToCollectionUpdate(status, boolField(item, "value", true))
			if err != nil {
				results = append(results, batchResult{Index: i, Input: label, Success: false, Error: err.Error()})
				continue
			}

			gameID, gameName, err := resolveGameID(client, item)
			if err != nil {
				results = append(results, batchResult{Index: i, Input: label, Success: false, Error: err.Error()})
				continue
			}
			if gameName != "" {
				label = gameName
			}

			if _, _, err := ws.Post("/geekcollection.php", gameReferer(gameID), buildCollectionPayload(gameID, update)); err != nil {
				results = append(results, batchResult{Index: i, Input: label, GameID: gameID, Success: false, Error: err.Error()})
				continue
			}

			results = append(results, batchResult{Index: i, Input: label, GameID: gameID, Success: true})
			succeeded++
		}

		summary := batchSummary{
			Total:     len(rawItems),
			Succeeded: succeeded,
			Failed:    len(rawItems) - succeeded,
			Results:   results,
		}
		out, err := json.Marshal(summary)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

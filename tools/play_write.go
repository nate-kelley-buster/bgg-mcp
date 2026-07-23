package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/faegents/bgg-mcp/session"
	"github.com/kkjdaniel/gogeek/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// playPayload is the POST body geekplay.php expects for creating or editing a
// play. CONFIRMED against a working reference implementation (a published
// open-source BGG integration) for the create case (PlayID omitted); the
// edit case (PlayID set) is presumed by symmetry with that confirmed shape,
// not independently verified. Note AJAX is an int here, unlike
// collectionPayload's string "1" — these two endpoints are not assumed to
// share conventions, since we only have concrete evidence for each
// separately.
type playPayload struct {
	Action     string       `json:"action"`
	ObjectID   int          `json:"objectid"`
	ObjectType string       `json:"objecttype"`
	PlayID     int          `json:"playid,omitempty"`
	PlayDate   string       `json:"playdate"`
	Length     int          `json:"length"`
	Comments   string       `json:"comments"`
	Location   string       `json:"location"`
	Incomplete bool         `json:"incomplete"`
	NoWinStats bool         `json:"nowinstats"`
	AJAX       int          `json:"ajax"`
	Quantity   int          `json:"quantity"`
	Players    []playPlayer `json:"players,omitempty"`
}

type playPlayer struct {
	Name     string `json:"name"`
	Win      bool   `json:"win"`
	Score    string `json:"score"`
	Position string `json:"position"`
	Color    string `json:"color"`
	Rating   int    `json:"rating"`
	New      bool   `json:"new"`
	Selected bool   `json:"selected"`
	Username string `json:"username,omitempty"`
}

// deletePlayPayload has no confirmed reference at all — it's the leanest
// plausible shape for BGG's play-deletion request.
type deletePlayPayload struct {
	Action string `json:"action"`
	PlayID int    `json:"playid"`
	AJAX   int    `json:"ajax"`
}

type playWriteResult struct {
	Success     bool   `json:"success"`
	GameID      int    `json:"game_id,omitempty"`
	GameName    string `json:"game_name,omitempty"`
	PlayID      int    `json:"play_id,omitempty"`
	RawResponse string `json:"raw_bgg_response"`
}

func stringField(m map[string]interface{}, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func boolField(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return def
}

func intField(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return def
}

func playDateOrToday(arguments map[string]interface{}) string {
	if date := stringField(arguments, "date", ""); date != "" {
		return date
	}
	return time.Now().Format("2006-01-02")
}

func gameLabel(name string, id int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("game %d", id)
}

// parsePlayPlayers converts the loosely-typed "players" argument (a JSON
// array of objects) into playPlayer structs. No schema is declared for the
// array's items in the tool definition — matching the existing codebase's
// preference for manual assertion over relying on the MCP schema layer for
// nested object shapes.
func parsePlayPlayers(raw interface{}) []playPlayer {
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}

	players := make([]playPlayer, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		players = append(players, playPlayer{
			Name:     stringField(m, "name", ""),
			Win:      boolField(m, "win", false),
			Score:    stringField(m, "score", ""),
			Position: stringField(m, "position", ""),
			Color:    stringField(m, "color", ""),
			Rating:   intField(m, "rating", 0),
			New:      true,
			Selected: true,
			Username: stringField(m, "username", ""),
		})
	}

	return players
}

func buildPlayPayload(arguments map[string]interface{}, gameID, playID int) playPayload {
	return playPayload{
		Action:     "save",
		ObjectID:   gameID,
		ObjectType: "thing",
		PlayID:     playID,
		PlayDate:   playDateOrToday(arguments),
		Length:     intField(arguments, "length", 0),
		Comments:   stringField(arguments, "comments", ""),
		Location:   stringField(arguments, "location", ""),
		Incomplete: boolField(arguments, "incomplete", false),
		NoWinStats: boolField(arguments, "nowinstats", false),
		AJAX:       1,
		Quantity:   intField(arguments, "quantity", 1),
		Players:    parsePlayPlayers(arguments["players"]),
	}
}

func postPlayPayload(ws *session.WriteSession, gameID int, gameName string, playID int, payload any) (*mcp.CallToolResult, error) {
	body, _, err := ws.Post("/geekplay.php", gameReferer(gameID), payload)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error saving play for %s: %v", gameLabel(gameName, gameID), err)), nil
	}

	result := playWriteResult{
		Success:     true,
		GameID:      gameID,
		GameName:    gameName,
		PlayID:      playID,
		RawResponse: string(body),
	}
	out, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func playToolParams() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("date", mcp.Description("Play date, YYYY-MM-DD. Defaults to today if omitted.")),
		mcp.WithNumber("quantity", mcp.Description("Number of times played (defaults to 1)"), mcp.Min(1)),
		mcp.WithNumber("length", mcp.Description("Play length in minutes (defaults to 0/unspecified)")),
		mcp.WithString("location", mcp.Description("Where the game was played")),
		mcp.WithString("comments", mcp.Description("Freeform comments about the play")),
		mcp.WithBoolean("incomplete", mcp.Description("Whether the play was incomplete (defaults to false)")),
		mcp.WithBoolean("nowinstats", mcp.Description("Exclude this play from win statistics (defaults to false)")),
		mcp.WithArray("players", mcp.Description("Players in this play. Each item: {name (string, required), win (bool), score (string), position (string), color (string), rating (number 1-10), username (string, optional BGG username)}")),
	}
}

func LogPlayTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("Log a play of a game to your BGG play history. CONFIRMED payload shape, verified against a working reference implementation — the most reliable of the new write tools."),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
	}, playToolParams()...)
	tool := mcp.NewTool("bgg-log-play", opts...)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		payload := buildPlayPayload(arguments, gameID, 0)
		return postPlayPayload(ws, gameID, gameName, 0, payload)
	}

	return tool, handler
}

func UpdatePlayTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("Edit an existing play on BGG by play id. UNVERIFIED: presumed by symmetry with the confirmed create-a-play payload to use the same 'save' action with an existing play id present — not confirmed against a live response."),
		mcp.WithNumber("play_id", mcp.Required(), mcp.Description("The BGG play id to edit")),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id' — required because the payload this is modeled on always includes the game.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name'.")),
	}, playToolParams()...)
	tool := mcp.NewTool("bgg-update-play", opts...)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		playIDVal, ok := arguments["play_id"].(float64)
		if !ok {
			return mcp.NewToolResultText("play_id is required"), nil
		}

		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		playID := int(playIDVal)
		payload := buildPlayPayload(arguments, gameID, playID)
		return postPlayPayload(ws, gameID, gameName, playID, payload)
	}

	return tool, handler
}

func DeletePlayTool(ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-delete-play",
		mcp.WithDescription("Delete a play from your BGG play history by play id. UNVERIFIED: no confirmed reference for this operation exists — this is the leanest plausible payload and may need objectid/objecttype added if BGG rejects a bare delete."),
		mcp.WithNumber("play_id", mcp.Required(), mcp.Description("The BGG play id to delete")),
		mcp.WithDestructiveHintAnnotation(true),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		playIDVal, ok := arguments["play_id"].(float64)
		if !ok {
			return mcp.NewToolResultText("play_id is required"), nil
		}
		playID := int(playIDVal)

		payload := deletePlayPayload{Action: "delete", PlayID: playID, AJAX: 1}
		body, _, err := ws.Post("/geekplay.php", "https://boardgamegeek.com/plays", payload)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error deleting play %d: %v", playID, err)), nil
		}

		result := playWriteResult{Success: true, PlayID: playID, RawResponse: string(body)}
		out, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/faegents/bgg-mcp/session"
	"github.com/kkjdaniel/gogeek/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// collectionPayload is the POST body geekcollection.php expects. Only
// AddOwned is confirmed against a real working reference implementation; the
// remaining Add* fields are inferred by naming-pattern symmetry — BGG's own
// XML API2 collection *read* response documents own/prevowned/fortrade/want/
// wanttoplay/wanttobuy/wishlist/preordered as the equivalent boolean flags —
// but have not been verified against a live write response. Rating/Comment
// follow the same inference from the documented <rating>/<comment> read
// fields. Treat every field but AddOwned as best-effort until live-tested.
type collectionPayload struct {
	ObjectType string `json:"objecttype"`
	ObjectID   int    `json:"objectid"`
	AJAX       string `json:"ajax"`
	Action     string `json:"action"`
	QuickAdd   string `json:"quickadd,omitempty"`

	AddOwned      string `json:"addowned,omitempty"`
	AddPrevOwned  string `json:"addprevowned,omitempty"`
	AddForTrade   string `json:"addfortrade,omitempty"`
	AddWant       string `json:"addwant,omitempty"`
	AddWantToPlay string `json:"addwanttoplay,omitempty"`
	AddWantToBuy  string `json:"addwanttobuy,omitempty"`
	AddWishlist   string `json:"addwishlist,omitempty"`
	AddPreordered string `json:"addpreordered,omitempty"`

	WishlistPriority int    `json:"wishlistpriority,omitempty"`
	Rating           string `json:"rating,omitempty"`
	Comment          string `json:"comment,omitempty"`
}

// collectionStatusUpdate carries only the fields a caller explicitly wants to
// change; nil means "leave as-is on BGG's side" (omitted from the JSON body).
type collectionStatusUpdate struct {
	Own, PrevOwned, ForTrade, Want, WantToPlay, WantToBuy, Wishlist, Preordered *bool
	WishlistPriority                                                            *int
	Rating                                                                      *float64
	Comment                                                                     *string
}

func buildCollectionPayload(gameID int, u collectionStatusUpdate) collectionPayload {
	p := collectionPayload{
		ObjectType: "thing",
		ObjectID:   gameID,
		AJAX:       "1",
		Action:     "additem",
		QuickAdd:   "1",
	}

	setFlag := func(dst *string, v *bool) {
		if v != nil {
			*dst = boolToBGGFlag(*v)
		}
	}
	setFlag(&p.AddOwned, u.Own)
	setFlag(&p.AddPrevOwned, u.PrevOwned)
	setFlag(&p.AddForTrade, u.ForTrade)
	setFlag(&p.AddWant, u.Want)
	setFlag(&p.AddWantToPlay, u.WantToPlay)
	setFlag(&p.AddWantToBuy, u.WantToBuy)
	setFlag(&p.AddWishlist, u.Wishlist)
	setFlag(&p.AddPreordered, u.Preordered)

	if u.WishlistPriority != nil {
		p.WishlistPriority = *u.WishlistPriority
	}
	if u.Rating != nil {
		p.Rating = strconv.FormatFloat(*u.Rating, 'f', -1, 64)
	}
	if u.Comment != nil {
		p.Comment = *u.Comment
	}

	return p
}

func boolToBGGFlag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// deleteCollectionItemPayload is an alternate, more aggressive removal
// attempt for if clearing all status flags (RemoveFromCollectionTool's
// default behavior) turns out to only "un-status" an item rather than delete
// its collection row entirely. Swap postCollectionUpdate's call in
// RemoveFromCollectionTool's handler for this if live testing shows the row
// survives with every flag false.
func deleteCollectionItemPayload(gameID int) collectionPayload {
	return collectionPayload{
		ObjectType: "thing",
		ObjectID:   gameID,
		AJAX:       "1",
		Action:     "deleteitem",
	}
}

// collectionWriteResult is the shared response shape for every
// geekcollection.php write. RawResponse is always included, not just on
// error, because this endpoint's real success/failure response shape is
// unverified — seeing BGG's actual response is how the payloads above get
// confirmed or corrected during live testing.
type collectionWriteResult struct {
	Success     bool   `json:"success"`
	GameID      int    `json:"game_id"`
	GameName    string `json:"game_name,omitempty"`
	RawResponse string `json:"raw_bgg_response"`
}

func postCollectionPayload(ws *session.WriteSession, gameID int, gameName string, payload collectionPayload) (*mcp.CallToolResult, error) {
	body, _, err := ws.Post("/geekcollection.php", gameReferer(gameID), payload)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error updating collection for game %d: %v", gameID, err)), nil
	}

	result := collectionWriteResult{
		Success:     true,
		GameID:      gameID,
		GameName:    gameName,
		RawResponse: string(body),
	}
	out, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error formatting result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func postCollectionUpdate(ws *session.WriteSession, gameID int, gameName string, update collectionStatusUpdate) (*mcp.CallToolResult, error) {
	return postCollectionPayload(ws, gameID, gameName, buildCollectionPayload(gameID, update))
}

func UpdateCollectionStatusTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-update-collection-status",
		mcp.WithDescription("Set collection status flags for a game in your BGG collection (own, previously owned, for trade, want, want to play, want to buy, wishlist + priority, preordered). Only flags you explicitly pass are changed; omitted flags are left as-is on BGG's side. Uses BGG's undocumented collection-write endpoint — the 'own' flag is confirmed working, other flags are inferred from BGG's documented field names and unverified until tested live."),
		mcp.WithString("name", mcp.Description("The game's name (resolved via BGG search). Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
		mcp.WithBoolean("own", mcp.Description("Mark as owned (true) or not owned (false)")),
		mcp.WithBoolean("prev_owned", mcp.Description("Mark as previously owned")),
		mcp.WithBoolean("for_trade", mcp.Description("Mark as for trade")),
		mcp.WithBoolean("want", mcp.Description("Mark as wanted")),
		mcp.WithBoolean("want_to_play", mcp.Description("Mark as wanted to play")),
		mcp.WithBoolean("want_to_buy", mcp.Description("Mark as wanted to buy")),
		mcp.WithBoolean("wishlist", mcp.Description("Add to (or remove from) wishlist")),
		mcp.WithBoolean("preordered", mcp.Description("Mark as preordered")),
		mcp.WithNumber("wishlist_priority", mcp.Description("Wishlist priority, 1 (must have) to 5 (least important)"), mcp.Min(1), mcp.Max(5)),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		update := collectionStatusUpdate{}
		if v, ok := arguments["own"].(bool); ok {
			update.Own = &v
		}
		if v, ok := arguments["prev_owned"].(bool); ok {
			update.PrevOwned = &v
		}
		if v, ok := arguments["for_trade"].(bool); ok {
			update.ForTrade = &v
		}
		if v, ok := arguments["want"].(bool); ok {
			update.Want = &v
		}
		if v, ok := arguments["want_to_play"].(bool); ok {
			update.WantToPlay = &v
		}
		if v, ok := arguments["want_to_buy"].(bool); ok {
			update.WantToBuy = &v
		}
		if v, ok := arguments["wishlist"].(bool); ok {
			update.Wishlist = &v
		}
		if v, ok := arguments["preordered"].(bool); ok {
			update.Preordered = &v
		}
		if v, ok := arguments["wishlist_priority"].(float64); ok {
			priority := int(v)
			update.WishlistPriority = &priority
		}

		return postCollectionUpdate(ws, gameID, gameName, update)
	}

	return tool, handler
}

func RemoveFromCollectionTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-remove-from-collection",
		mcp.WithDescription("Remove a game from your BGG collection entirely. UNVERIFIED: there is no confirmed reference for this operation — the current implementation clears every collection status flag, which may only 'un-status' the item rather than delete the collection row. Treat as best-effort until confirmed live."),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
		mcp.WithDestructiveHintAnnotation(true),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		f := false
		update := collectionStatusUpdate{
			Own: &f, PrevOwned: &f, ForTrade: &f, Want: &f,
			WantToPlay: &f, WantToBuy: &f, Wishlist: &f, Preordered: &f,
		}

		return postCollectionUpdate(ws, gameID, gameName, update)
	}

	return tool, handler
}

func RateGameTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-rate-game",
		mcp.WithDescription("Set your rating for a game on BGG (1-10, decimals allowed). UNVERIFIED: inferred from BGG's documented rating field, not confirmed against a live response."),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
		mcp.WithNumber("rating", mcp.Required(), mcp.Description("Your rating for the game, 1-10"), mcp.Min(1), mcp.Max(10)),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		rating, ok := arguments["rating"].(float64)
		if !ok {
			return mcp.NewToolResultText("rating is required"), nil
		}

		update := collectionStatusUpdate{Rating: &rating}
		return postCollectionUpdate(ws, gameID, gameName, update)
	}

	return tool, handler
}

func SetCollectionNotesTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-set-collection-notes",
		mcp.WithDescription("Set the private notes/comment field on a game in your BGG collection. UNVERIFIED: inferred from BGG's documented comment field, not confirmed against a live response."),
		mcp.WithString("name", mcp.Description("The game's name. Provide this or 'id', not both.")),
		mcp.WithNumber("id", mcp.Description("The game's BGG id. Provide this or 'name', not both.")),
		mcp.WithString("notes", mcp.Required(), mcp.Description("The private note/comment text to set")),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if result, done := requireWriteSession(ws); done {
			return result, nil
		}

		arguments := request.GetArguments()
		gameID, gameName, err := resolveGameID(client, arguments)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error resolving game: %v", err)), nil
		}

		notes, ok := arguments["notes"].(string)
		if !ok {
			return mcp.NewToolResultText("notes is required"), nil
		}

		update := collectionStatusUpdate{Comment: &notes}
		return postCollectionUpdate(ws, gameID, gameName, update)
	}

	return tool, handler
}

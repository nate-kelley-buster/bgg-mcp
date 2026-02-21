package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kkjdaniel/gogeek/v2"
	gofamily "github.com/kkjdaniel/gogeek/v2/family"
	"github.com/kkjdaniel/gogeek/v2/thing"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type GameFamilyInfo struct {
	FamilyID   int    `json:"family_id"`
	FamilyName string `json:"family_name"`
}

type GameFamilyResult struct {
	GameID   int              `json:"game_id"`
	GameName string           `json:"game_name"`
	Families []GameFamilyInfo `json:"families"`
}

type FamilyDetailResult struct {
	FamilyID    int                `json:"family_id"`
	FamilyName  string             `json:"family_name"`
	Description string             `json:"description"`
	MemberGames []FamilyMemberGame `json:"member_games"`
}

type FamilyMemberGame struct {
	GameID   int    `json:"game_id"`
	GameName string `json:"game_name"`
}

func GameFamilyTool(client *gogeek.Client) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-game-family",
		mcp.WithDescription("Look up board game families (series/universes) on BoardGameGeek. Use 'name' or 'id' to find which families a game belongs to. Use 'family_id' to get all games in a specific family/series."),
		mcp.WithString("name",
			mcp.Description("Name of the board game to look up families for."),
		),
		mcp.WithNumber("id",
			mcp.Description("BGG ID of the board game to look up families for. Preferred over 'name' when already known."),
		),
		mcp.WithNumber("family_id",
			mcp.Description("BGG family ID to look up directly. Returns all games in that family/series."),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Mode 1: look up a specific family by family_id
		if familyIDVal, ok := arguments["family_id"]; ok && familyIDVal != nil {
			var familyID int
			switch v := familyIDVal.(type) {
			case float64:
				familyID = int(v)
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return mcp.NewToolResultText("Invalid family_id format"), nil
				}
				familyID = parsed
			default:
				return mcp.NewToolResultText("Invalid family_id type"), nil
			}

			familyData, err := gofamily.Query(client, familyID, gofamily.BoardGameFamily)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Error fetching family %d: %v", familyID, err)), nil
			}

			if len(familyData.Items) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("No family found with ID %d", familyID)), nil
			}

			fam := familyData.Items[0]
			result := FamilyDetailResult{
				FamilyID:    fam.ID,
				FamilyName:  fam.Name.Value,
				Description: fam.Description,
				MemberGames: []FamilyMemberGame{},
			}

			for _, link := range fam.Links {
				if link.Inbound {
					result.MemberGames = append(result.MemberGames, FamilyMemberGame{
						GameID:   link.ID,
						GameName: link.Value,
					})
				}
			}

			out, err := json.Marshal(result)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Error formatting results: %v", err)), nil
			}
			return mcp.NewToolResultText(string(out)), nil
		}

		// Mode 2: look up families for a game (by name or ID)
		var gameID int

		if idVal, ok := arguments["id"]; ok && idVal != nil {
			switch v := idVal.(type) {
			case float64:
				gameID = int(v)
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return mcp.NewToolResultText("Invalid id format"), nil
				}
				gameID = parsed
			default:
				return mcp.NewToolResultText("Invalid id type"), nil
			}
		} else if nameVal, ok := arguments["name"].(string); ok && nameVal != "" {
			bestMatch, err := findBestGameMatch(client, nameVal)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Failed to find game: %v", err)), nil
			}
			gameID = bestMatch.ID
		} else {
			return mcp.NewToolResultText("Provide one of: 'name', 'id' (to find a game's families), or 'family_id' (to list all games in a family)"), nil
		}

		things, err := thing.Query(client, []int{gameID})
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error fetching game details: %v", err)), nil
		}

		if len(things.Items) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No game found with ID %d", gameID)), nil
		}

		gameItem := things.Items[0]

		gameName := ""
		if len(gameItem.Name) > 0 {
			gameName = gameItem.Name[0].Value
		}

		result := GameFamilyResult{
			GameID:   gameItem.ID,
			GameName: gameName,
			Families: []GameFamilyInfo{},
		}

		for _, link := range gameItem.Links {
			if link.Type == "boardgamefamily" {
				result.Families = append(result.Families, GameFamilyInfo{
					FamilyID:   link.ID,
					FamilyName: link.Value,
				})
			}
		}

		out, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error formatting results: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

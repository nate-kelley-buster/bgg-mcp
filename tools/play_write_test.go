package tools

import (
	"testing"
	"time"
)

func TestBuildPlayPayload_Defaults(t *testing.T) {
	payload := buildPlayPayload(map[string]interface{}{}, 224517, 0)

	if payload.Action != "save" || payload.ObjectID != 224517 || payload.ObjectType != "thing" {
		t.Errorf("base fields wrong: %+v", payload)
	}
	if payload.AJAX != 1 {
		t.Errorf("AJAX = %d, want 1 (an int here, unlike collectionPayload's string \"1\")", payload.AJAX)
	}
	if payload.Quantity != 1 {
		t.Errorf("Quantity default = %d, want 1", payload.Quantity)
	}
	wantDate := time.Now().Format("2006-01-02")
	if payload.PlayDate != wantDate {
		t.Errorf("PlayDate default = %q, want today (%q)", payload.PlayDate, wantDate)
	}
	if payload.PlayID != 0 {
		t.Errorf("PlayID should be 0 (omitted) for a new play, got %d", payload.PlayID)
	}
}

func TestBuildPlayPayload_ExplicitFields(t *testing.T) {
	args := map[string]interface{}{
		"date":       "2026-01-15",
		"quantity":   float64(2),
		"length":     float64(90),
		"location":   "My House",
		"comments":   "Great game",
		"incomplete": true,
		"nowinstats": true,
	}
	payload := buildPlayPayload(args, 1, 999)

	if payload.PlayDate != "2026-01-15" {
		t.Errorf("PlayDate = %q, want 2026-01-15", payload.PlayDate)
	}
	if payload.Quantity != 2 || payload.Length != 90 {
		t.Errorf("Quantity/Length = %d/%d, want 2/90", payload.Quantity, payload.Length)
	}
	if payload.Location != "My House" || payload.Comments != "Great game" {
		t.Errorf("Location/Comments not set: %+v", payload)
	}
	if !payload.Incomplete || !payload.NoWinStats {
		t.Errorf("Incomplete/NoWinStats not set: %+v", payload)
	}
	if payload.PlayID != 999 {
		t.Errorf("PlayID = %d, want 999 (edit case)", payload.PlayID)
	}
}

func TestParsePlayPlayers(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"name":     "Alice",
			"win":      true,
			"score":    "42",
			"position": "1",
			"username": "alice_bgg",
		},
		map[string]interface{}{
			"name": "Bob",
		},
	}

	players := parsePlayPlayers(raw)
	if len(players) != 2 {
		t.Fatalf("got %d players, want 2", len(players))
	}
	if players[0].Name != "Alice" || !players[0].Win || players[0].Score != "42" || players[0].Username != "alice_bgg" {
		t.Errorf("player 0 wrong: %+v", players[0])
	}
	if !players[0].New || !players[0].Selected {
		t.Errorf("player 0 should default New/Selected to true: %+v", players[0])
	}
	if players[1].Name != "Bob" || players[1].Win {
		t.Errorf("player 1 wrong: %+v", players[1])
	}
}

func TestParsePlayPlayers_NilForNonArray(t *testing.T) {
	if got := parsePlayPlayers("not an array"); got != nil {
		t.Errorf("expected nil for non-array input, got %+v", got)
	}
	if got := parsePlayPlayers(nil); got != nil {
		t.Errorf("expected nil for nil input, got %+v", got)
	}
}

func TestPlayDateOrToday(t *testing.T) {
	withDate := map[string]interface{}{"date": "2020-05-01"}
	if got := playDateOrToday(withDate); got != "2020-05-01" {
		t.Errorf("playDateOrToday with explicit date = %q, want 2020-05-01", got)
	}

	want := time.Now().Format("2006-01-02")
	if got := playDateOrToday(map[string]interface{}{}); got != want {
		t.Errorf("playDateOrToday default = %q, want %q", got, want)
	}
}

func TestGameLabel(t *testing.T) {
	if got := gameLabel("Wingspan", 266192); got != "Wingspan" {
		t.Errorf("gameLabel with name = %q, want Wingspan", got)
	}
	if got := gameLabel("", 266192); got != "game 266192" {
		t.Errorf("gameLabel without name = %q, want \"game 266192\"", got)
	}
}

package tools

import (
	"testing"

	"github.com/faegents/bgg-mcp/session"
	"github.com/kkjdaniel/gogeek/v2"
)

func TestResolveGameID_ExplicitID(t *testing.T) {
	// nil client is safe here: the id-provided path never touches it.
	id, name, err := resolveGameID(nil, map[string]interface{}{"id": float64(266192)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 266192 {
		t.Errorf("id = %d, want 266192", id)
	}
	if name != "" {
		t.Errorf("name should be empty when resolved by id (no search performed), got %q", name)
	}
}

func TestResolveGameID_MissingBoth(t *testing.T) {
	if _, _, err := resolveGameID(nil, map[string]interface{}{}); err == nil {
		t.Error("expected error when neither id nor name is provided")
	}
}

func TestResolveGameID_ZeroIDFallsThroughToNameCheck(t *testing.T) {
	if _, _, err := resolveGameID(nil, map[string]interface{}{"id": float64(0)}); err == nil {
		t.Error("expected error: id=0 should not short-circuit, and no name was given")
	}
}

func TestGameReferer(t *testing.T) {
	want := "https://boardgamegeek.com/boardgame/266192"
	if got := gameReferer(266192); got != want {
		t.Errorf("gameReferer(266192) = %q, want %q", got, want)
	}
}

func TestRequireWriteSession_NilSession(t *testing.T) {
	if _, done := requireWriteSession(nil); !done {
		t.Error("nil session should require write session (done=true)")
	}
}

func TestRequireWriteSession_DisabledSession(t *testing.T) {
	ws := session.NewWriteSession(gogeek.NewClient(), "", "")
	if _, done := requireWriteSession(ws); !done {
		t.Error("session without credentials should require write session (done=true)")
	}
}

func TestRequireWriteSession_EnabledSession(t *testing.T) {
	ws := session.NewWriteSession(gogeek.NewClient(), "user", "pass")
	if result, done := requireWriteSession(ws); done {
		t.Errorf("session with credentials should not require write session, got done=true, result=%v", result)
	}
}

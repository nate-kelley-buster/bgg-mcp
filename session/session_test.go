package session

import (
	"testing"

	"github.com/kkjdaniel/gogeek/v2"
)

func TestLooksExpired(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		body        string
		want        bool
	}{
		{"2xx json success", 200, "application/json", `{"ok":true}`, false},
		{"2xx no content type, non-html body", 200, "", `{"ok":true}`, false},
		{"2xx empty body", 204, "", "", false},
		{"401 unauthorized", 401, "application/json", `{"error":"unauthorized"}`, true},
		{"500 server error", 500, "text/plain", "internal error", true},
		{"html login redirect by content type", 200, "text/html; charset=utf-8", "<html><body>login</body></html>", true},
		{"body starts with angle bracket, no content-type signal", 200, "", "<html>whatever</html>", true},
		{"body has leading whitespace before angle bracket", 200, "", "   <html></html>", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := looksExpired(tc.status, tc.contentType, []byte(tc.body))
			if got != tc.want {
				t.Errorf("looksExpired(%d, %q, %q) = %v, want %v", tc.status, tc.contentType, tc.body, got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate([]byte("hello"), 10); got != "hello" {
		t.Errorf("truncate should leave short input unchanged, got %q", got)
	}

	got := truncate([]byte("hello world"), 5)
	want := "hello... (truncated)"
	if got != want {
		t.Errorf("truncate(%q, 5) = %q, want %q", "hello world", got, want)
	}
}

func TestNewWriteSession_EnabledReflectsCredentials(t *testing.T) {
	client := gogeek.NewClient()

	if ws := NewWriteSession(client, "", ""); ws.Enabled() {
		t.Error("session with no username/password should not be enabled")
	}
	if ws := NewWriteSession(client, "user", ""); ws.Enabled() {
		t.Error("session with only a username should not be enabled")
	}
	if ws := NewWriteSession(client, "", "pass"); ws.Enabled() {
		t.Error("session with only a password should not be enabled")
	}
	if ws := NewWriteSession(client, "user", "pass"); !ws.Enabled() {
		t.Error("session with both username and password should be enabled")
	}
}

func TestPost_DisabledSessionFailsWithoutNetworkCall(t *testing.T) {
	ws := NewWriteSession(gogeek.NewClient(), "", "")
	if _, _, err := ws.Post("/geekcollection.php", "https://boardgamegeek.com", nil); err == nil {
		t.Error("Post on a disabled session should error without attempting a request")
	}
}

func TestCookieHeader_DisabledSessionFailsWithoutNetworkCall(t *testing.T) {
	ws := NewWriteSession(gogeek.NewClient(), "", "")
	if _, err := ws.CookieHeader(); err == nil {
		t.Error("CookieHeader on a disabled session should error without attempting a login")
	}
}

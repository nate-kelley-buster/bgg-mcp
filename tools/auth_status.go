package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/faegents/bgg-mcp/session"
	"github.com/kkjdaniel/gogeek/v2"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// authStatusResult is intentionally diagnostic-only: SessionCookieNames lists
// which cookies a successful login produced, never their values, since a
// session cookie is itself a live credential and shouldn't be echoed back
// through a tool result.
type authStatusResult struct {
	ReadAuthMode        string   `json:"read_auth_mode"`
	WriteSessionEnabled bool     `json:"write_session_enabled"`
	LoginAttempted      bool     `json:"login_attempted"`
	LoginSucceeded      bool     `json:"login_succeeded"`
	LoginError          string   `json:"login_error,omitempty"`
	SessionCookieNames  []string `json:"session_cookie_names,omitempty"`
}

func authModeString(mode gogeek.AuthMode) string {
	switch mode {
	case gogeek.AuthAPIKey:
		return "api_key"
	case gogeek.AuthCookie:
		return "cookie"
	default:
		return "none"
	}
}

func cookieNames(cookieHeader string) []string {
	parts := strings.Split(cookieHeader, ";")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, _, _ := strings.Cut(p, "=")
		names = append(names, name)
	}
	return names
}

func AuthStatusTool(client *gogeek.Client, ws *session.WriteSession) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("bgg-auth-status",
		mcp.WithDescription("Diagnostic tool: reports the read client's auth mode (api_key/cookie/none) and, if BGG_USERNAME/BGG_PASSWORD are set, attempts the write-session login and reports whether it succeeded. Useful for debugging unexpected 401s or empty results without access to the server's own logs."),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result := authStatusResult{
			ReadAuthMode:        authModeString(client.AuthMode()),
			WriteSessionEnabled: ws != nil && ws.Enabled(),
		}

		if result.WriteSessionEnabled {
			result.LoginAttempted = true
			cookie, err := ws.CookieHeader()
			if err != nil {
				result.LoginError = err.Error()
			} else {
				result.LoginSucceeded = true
				result.SessionCookieNames = cookieNames(cookie)
			}
		}

		out, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(err.Error()), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}

	return tool, handler
}

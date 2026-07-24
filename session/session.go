// Package session implements authenticated write access to BoardGameGeek.
//
// BGG's public XML API2 is read-only. Collection and play mutations go through
// undocumented endpoints BGG's own website uses internally (geekcollection.php,
// geekplay.php), authenticated by a logged-in session cookie rather than an API
// key. WriteSession owns that login flow and the resulting cookie jar.
package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/kkjdaniel/gogeek/v2"
	"go.uber.org/ratelimit"
)

const (
	bggBaseURL = "https://boardgamegeek.com"
	loginPath  = "/login/api/v1"

	// userAgent mimics a real browser. BGG's write endpoints aren't part of the
	// documented API and may reject requests that look scripted.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// WriteSession manages a logged-in BGG session for write operations. Login is
// lazy — the first Post call triggers it — so constructing a WriteSession (and
// running bgg-mcp without BGG_PASSWORD set) never touches the network.
type WriteSession struct {
	mu       sync.Mutex
	username string
	password string
	http     *http.Client
	limiter  ratelimit.Limiter
	loggedIn bool
}

// NewWriteSession returns a WriteSession bound to client's rate limiter, so
// reads and writes share a single request budget against BGG. If username or
// password is empty, Enabled reports false and Post always fails fast without
// making a request.
func NewWriteSession(client *gogeek.Client, username, password string) *WriteSession {
	jar, _ := cookiejar.New(nil)
	return &WriteSession{
		username: username,
		password: password,
		http:     &http.Client{Jar: jar},
		limiter:  client.Limiter(),
	}
}

// Enabled reports whether this session has credentials to attempt writes.
func (s *WriteSession) Enabled() bool {
	return s.username != "" && s.password != ""
}

// Post sends an authenticated JSON POST to path (relative to boardgamegeek.com)
// with the given referer, logging in first if needed and retrying once after a
// fresh login if the response looks like an expired/invalid session. The raw
// response body is always returned alongside any error so a caller can inspect
// BGG's actual response — these endpoints are unofficial and undocumented, so
// diagnosing a rejected payload requires seeing exactly what BGG sent back.
func (s *WriteSession) Post(path, referer string, payload any) ([]byte, int, error) {
	if !s.Enabled() {
		return nil, 0, fmt.Errorf("write operations require BGG_USERNAME and BGG_PASSWORD to be set")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.loggedIn {
		if err := s.login(); err != nil {
			return nil, 0, fmt.Errorf("login failed: %w", err)
		}
	}

	body, status, contentType, err := s.doPost(path, referer, payload)
	if err != nil {
		return nil, 0, err
	}

	if looksExpired(status, contentType, body) {
		s.loggedIn = false
		if err := s.login(); err != nil {
			return nil, 0, fmt.Errorf("session expired and re-login failed: %w", err)
		}
		body, status, contentType, err = s.doPost(path, referer, payload)
		if err != nil {
			return nil, 0, err
		}
		if looksExpired(status, contentType, body) {
			return body, status, fmt.Errorf("BGG rejected the request after re-login (status %d): %s", status, truncate(body, 500))
		}
	}

	if status < 200 || status >= 300 {
		return body, status, fmt.Errorf("BGG returned status %d: %s", status, truncate(body, 500))
	}

	return body, status, nil
}

// CookieHeader ensures a login has happened and returns the resulting
// session's Cookie header value for boardgamegeek.com. This lets a caller
// authenticate read-only XML API2 requests (via gogeek's WithCookie option)
// using the exact same username/password login this package already
// performs for writes — so a user who only has BGG_USERNAME/BGG_PASSWORD
// never needs a separate BGG_API_KEY at all.
func (s *WriteSession) CookieHeader() (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("BGG_USERNAME and BGG_PASSWORD must be set")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.loggedIn {
		if err := s.login(); err != nil {
			return "", fmt.Errorf("login failed: %w", err)
		}
	}

	base, err := url.Parse(bggBaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse BGG base URL: %w", err)
	}

	cookies := s.http.Jar.Cookies(base)
	if len(cookies) == 0 {
		return "", fmt.Errorf("login succeeded but no session cookies were set")
	}

	parts := make([]string, len(cookies))
	for i, c := range cookies {
		parts[i] = c.Name + "=" + c.Value
	}
	return strings.Join(parts, "; "), nil
}

func (s *WriteSession) login() error {
	s.limiter.Take()

	buf, err := json.Marshal(map[string]any{
		"credentials": map[string]string{
			"username": s.username,
			"password": s.password,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to build login payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, bggBaseURL+loginPath, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("failed to build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", bggBaseURL+"/login")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("BGG login returned status %d: %s", resp.StatusCode, truncate(respBody, 500))
	}

	s.loggedIn = true
	return nil
}

func (s *WriteSession) doPost(path, referer string, payload any) (body []byte, status int, contentType string, err error) {
	s.limiter.Take()

	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to build request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, bggBaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to read response from %s: %w", path, err)
	}

	return respBody, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

// looksExpired flags a response as an expired/invalid session rather than a
// legitimate (possibly error) reply from the write endpoint itself. It's
// deliberately conservative: these endpoints are unofficial and undocumented,
// so we don't know their exact success response shape, but a non-2xx status or
// an HTML page (BGG's login-redirect behavior) are strong, low-risk signals.
func looksExpired(status int, contentType string, body []byte) bool {
	if status < 200 || status >= 300 {
		return true
	}
	if strings.Contains(strings.ToLower(contentType), "html") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	return len(trimmed) > 0 && trimmed[0] == '<'
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "... (truncated)"
}

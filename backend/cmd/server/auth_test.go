package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"golang.org/x/oauth2"
)

type stubAuthStore struct {
	upsertedUsers   []authenticatedUser
	createdSessions []authSessionRecord
	deletedTokens   []string
	sessionUser     database.SessionUserRecord
	upsertedUser    authenticatedUser
	sessionErr      error
	upsertErr       error
	createErr       error
	deleteErr       error
}

func (s *stubAuthStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	if s.upsertErr != nil {
		return authenticatedUser{}, s.upsertErr
	}
	s.upsertedUsers = append(s.upsertedUsers, user)
	if s.upsertedUser.ID != "" {
		return s.upsertedUser, nil
	}
	return user, nil
}

func (s *stubAuthStore) CreateSession(_ context.Context, session authSessionRecord) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createdSessions = append(s.createdSessions, session)
	return nil
}

func (s *stubAuthStore) GetSessionUserByToken(_ context.Context, sessionToken string, _ time.Time) (database.SessionUserRecord, error) {
	if s.sessionErr != nil {
		return database.SessionUserRecord{}, s.sessionErr
	}
	if sessionToken == "" {
		return database.SessionUserRecord{}, database.ErrSessionNotFound
	}
	return s.sessionUser, nil
}

func (s *stubAuthStore) DeleteSession(_ context.Context, sessionToken string) error {
	s.deletedTokens = append(s.deletedTokens, sessionToken)
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return nil
}

func (s *stubAuthStore) Close() error { return nil }

func TestAuthenticatedUserHelpers(t *testing.T) {
	t.Run("is authenticated when id exists", func(t *testing.T) {
		if !(authenticatedUser{ID: "user-1"}).isAuthenticated() {
			t.Fatal("isAuthenticated() = false; want true")
		}
	})

	t.Run("not authenticated when id missing", func(t *testing.T) {
		if (authenticatedUser{ID: "   "}).isAuthenticated() {
			t.Fatal("isAuthenticated() = true; want false")
		}
	})

	t.Run("display name prefers name", func(t *testing.T) {
		got := (authenticatedUser{ID: "user-1", Name: "  Jane Doe  ", Email: "jane@example.com"}).displayName()
		if got != "Jane Doe" {
			t.Fatalf("displayName() = %q; want Jane Doe", got)
		}
	})

	t.Run("display name falls back to email", func(t *testing.T) {
		got := (authenticatedUser{ID: "user-1", Email: "  jane@example.com  "}).displayName()
		if got != "jane@example.com" {
			t.Fatalf("displayName() = %q; want jane@example.com", got)
		}
	})

	t.Run("display name falls back to id", func(t *testing.T) {
		got := (authenticatedUser{ID: "  user-1  "}).displayName()
		if got != "user-1" {
			t.Fatalf("displayName() = %q; want user-1", got)
		}
	})
}

func TestSessionFromRequest(t *testing.T) {
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)

	t.Run("requires cookie", func(t *testing.T) {
		handler := &authHandler{store: &stubAuthStore{}, now: func() time.Time { return now }}
		_, err := handler.sessionFromRequest(httptest.NewRequest(http.MethodGet, "/auth/session", nil))
		if !errors.Is(err, errAuthenticationRequired) {
			t.Fatalf("sessionFromRequest() error = %v; want errAuthenticationRequired", err)
		}
	})

	t.Run("returns authenticated session from store", func(t *testing.T) {
		store := &stubAuthStore{sessionUser: database.SessionUserRecord{
			ID:        "user-123",
			Name:      "Player One",
			Email:     "player@example.com",
			ImageURL:  "https://cdn.example.com/player.png",
			ExpiresAt: now.Add(time.Hour),
		}}
		handler := &authHandler{store: store, now: func() time.Time { return now }}
		request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})

		session, err := handler.sessionFromRequest(request)
		if err != nil {
			t.Fatalf("sessionFromRequest() error = %v", err)
		}
		if !session.valid || session.user.ID != "user-123" || session.token != "session-token" {
			t.Fatalf("session = %#v; want valid authenticated session", session)
		}
	})

	t.Run("maps missing session to authentication required", func(t *testing.T) {
		store := &stubAuthStore{sessionErr: database.ErrSessionNotFound}
		handler := &authHandler{store: store, now: func() time.Time { return now }}
		request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})

		_, err := handler.sessionFromRequest(request)
		if !errors.Is(err, errAuthenticationRequired) {
			t.Fatalf("sessionFromRequest() error = %v; want errAuthenticationRequired", err)
		}
	})

	t.Run("returns unexpected store error", func(t *testing.T) {
		store := &stubAuthStore{sessionErr: errors.New("session store boom")}
		handler := &authHandler{store: store, now: func() time.Time { return now }}
		request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})

		_, err := handler.sessionFromRequest(request)
		if err == nil || err.Error() != "session store boom" {
			t.Fatalf("sessionFromRequest() error = %v; want session store boom", err)
		}
	})
}

func TestSessionTokenForUserReplacesDifferentExistingSession(t *testing.T) {
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	store := &stubAuthStore{
		sessionUser: database.SessionUserRecord{
			ID:        "old-user",
			Name:      "Old User",
			Email:     "old@example.com",
			ExpiresAt: now.Add(time.Hour),
		},
		deleteErr: errors.New("delete boom"),
	}
	handler := &authHandler{store: store, now: func() time.Time { return now }}
	request := httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "old-token"})

	token, err := handler.sessionTokenForUser(request, authenticatedUser{ID: "new-user"}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("sessionTokenForUser() error = %v", err)
	}
	if token == "" || token == "old-token" {
		t.Fatalf("sessionTokenForUser() token = %q; want fresh token", token)
	}
	if len(store.deletedTokens) != 1 || store.deletedTokens[0] != "old-token" {
		t.Fatalf("deletedTokens = %#v; want old-token", store.deletedTokens)
	}
	if len(store.createdSessions) != 1 || store.createdSessions[0].UserID != "new-user" {
		t.Fatalf("createdSessions = %#v; want new-user session", store.createdSessions)
	}
}

func TestSessionTokenForUserReturnsUnexpectedSessionError(t *testing.T) {
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	store := &stubAuthStore{sessionErr: errors.New("session lookup boom")}
	handler := &authHandler{store: store, now: func() time.Time { return now }}
	request := httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "old-token"})

	_, err := handler.sessionTokenForUser(request, authenticatedUser{ID: "new-user"}, now.Add(time.Hour))
	if err == nil || err.Error() != "session lookup boom" {
		t.Fatalf("sessionTokenForUser() error = %v; want session lookup boom", err)
	}
}

func TestHandleSession(t *testing.T) {
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	t.Setenv("FRONTEND_URL", "http://frontend.test")

	t.Run("returns null for unauthenticated request", func(t *testing.T) {
		handler := &authHandler{store: &stubAuthStore{}, now: func() time.Time { return now }}
		request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		response := httptest.NewRecorder()

		handler.handleSession(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("handleSession() status = %d; want 200", response.Code)
		}
		if body := response.Body.String(); body != "null" {
			t.Fatalf("handleSession() body = %q; want null", body)
		}
		if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Fatalf("Access-Control-Allow-Credentials = %q; want true", got)
		}
	})

	t.Run("returns session payload for authenticated request", func(t *testing.T) {
		store := &stubAuthStore{sessionUser: database.SessionUserRecord{
			ID:        "user-123",
			Name:      "Player One",
			Email:     "player@example.com",
			ImageURL:  "https://cdn.example.com/player.png",
			ExpiresAt: now.Add(time.Hour),
		}}
		handler := &authHandler{store: store, now: func() time.Time { return now }}
		request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		response := httptest.NewRecorder()

		handler.handleSession(response, request)

		var payload sessionResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.User.ID != "user-123" || payload.User.Email != "player@example.com" {
			t.Fatalf("payload = %#v; want authenticated session payload", payload)
		}
	})
}

func TestHandleLogout(t *testing.T) {
	handler := &authHandler{store: &stubAuthStore{}}
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	handler.handleLogout(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("handleLogout() status = %d; want 204", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) < 3 {
		t.Fatalf("logout cookies = %d; want at least 3 clears", len(cookies))
	}
}

func TestHandleGoogleSignInSetsStateAndPKCECookies(t *testing.T) {
	handler := &authHandler{
		config: authConfig{
			cookieDomain: "kompozicijas.xyz",
			oauthConfig: &oauth2.Config{
				Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth"},
			},
		},
		now:   func() time.Time { return time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC) },
		state: func() (string, error) { return "state-token", nil },
	}
	request := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	response := httptest.NewRecorder()

	handler.handleGoogleSignIn(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("handleGoogleSignIn() status = %d; want 302", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies = %d; want 2", len(cookies))
	}
	redirectURL, err := url.Parse(response.Result().Header.Get("Location"))
	if err != nil {
		t.Fatalf("url.Parse(Location) error = %v", err)
	}
	query := redirectURL.Query()
	if query.Get("state") != "state-token" {
		t.Fatalf("state query = %q; want state-token", query.Get("state"))
	}
	if query.Get("code_challenge") == "" {
		t.Fatal("code_challenge = empty; want PKCE challenge")
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q; want S256", query.Get("code_challenge_method"))
	}
	if cookies[0].Name != oauthStateCookieName && cookies[1].Name != oauthStateCookieName {
		t.Fatal("oauth state cookie not set")
	}
	if cookies[0].Name != oauthPKCECookieName && cookies[1].Name != oauthPKCECookieName {
		t.Fatal("oauth pkce cookie not set")
	}
	for _, cookie := range cookies {
		if cookie.Domain != "kompozicijas.xyz" {
			t.Fatalf("cookie %q domain = %q; want kompozicijas.xyz", cookie.Name, cookie.Domain)
		}
	}
}

type failingRandomReader struct{}

func (failingRandomReader) Read([]byte) (int, error) {
	return 0, errors.New("random failure")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newOAuthHTTPClient(tokenErr, userInfoErr error, tokenStatus int, tokenBody, userInfoBody string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case "https://oauth.test/token":
			if tokenErr != nil {
				return nil, tokenErr
			}
			return jsonHTTPResponse(tokenStatus, tokenBody), nil
		case googleUserInfoEndpoint:
			if userInfoErr != nil {
				return nil, userInfoErr
			}
			return jsonHTTPResponse(http.StatusOK, userInfoBody), nil
		default:
			return nil, fmt.Errorf("unexpected request url %s", request.URL)
		}
	})}
}

func newOAuthCallbackHandler(store *stubAuthStore, now time.Time) *authHandler {
	return &authHandler{
		config: authConfig{
			oauthConfig: &oauth2.Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://oauth.test/auth",
					TokenURL: "https://oauth.test/token",
				},
			},
			frontendURL:    "http://frontend.test/",
			frontendOrigin: "http://frontend.test",
			sessionTTL:     2 * time.Hour,
		},
		store: store,
		now:   func() time.Time { return now },
		state: func() (string, error) { return "state-token", nil },
	}
}

func newOAuthCallbackRequest(rawQuery string, client *http.Client) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?"+rawQuery, nil)
	request.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: stateDigest("state-token")})
	request.AddCookie(&http.Cookie{Name: oauthPKCECookieName, Value: "pkce-verifier"})
	return request.WithContext(context.WithValue(request.Context(), oauth2.HTTPClient, client))
}

func cookieValue(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func TestAuthConfigHelpers(t *testing.T) {
	t.Run("google oauth config env validation", func(t *testing.T) {
		t.Setenv("GOOGLE_CLIENT_ID", "")
		if _, err := newGoogleOAuthConfigFromEnv(); err == nil || err.Error() != "GOOGLE_CLIENT_ID is required" {
			t.Fatalf("newGoogleOAuthConfigFromEnv() error = %v; want GOOGLE_CLIENT_ID is required", err)
		}

		t.Setenv("GOOGLE_CLIENT_ID", "client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "")
		if _, err := newGoogleOAuthConfigFromEnv(); err == nil || err.Error() != "GOOGLE_CLIENT_SECRET is required" {
			t.Fatalf("newGoogleOAuthConfigFromEnv() error = %v; want GOOGLE_CLIENT_SECRET is required", err)
		}

		t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
		t.Setenv("BASE_URL", "frontend.test")
		if _, err := newGoogleOAuthConfigFromEnv(); err == nil || err.Error() != "BASE_URL must be a valid absolute URL" {
			t.Fatalf("newGoogleOAuthConfigFromEnv() error = %v; want BASE_URL must be a valid absolute URL", err)
		}

		t.Setenv("BASE_URL", "https://backend.test/")
		config, err := newGoogleOAuthConfigFromEnv()
		if err != nil {
			t.Fatalf("newGoogleOAuthConfigFromEnv() error = %v", err)
		}
		if config.RedirectURL != "https://backend.test/auth/google/callback" {
			t.Fatalf("RedirectURL = %q; want https://backend.test/auth/google/callback", config.RedirectURL)
		}
	})

	t.Run("configured auth handler validates inputs and secure cookie", func(t *testing.T) {
		if _, err := newConfiguredAuthHandler(nil); err == nil || err.Error() != "auth store is required" {
			t.Fatalf("newConfiguredAuthHandler(nil) error = %v; want auth store is required", err)
		}

		t.Setenv("BASE_URL", "https://backend.test")
		t.Setenv("FRONTEND_URL", "https://frontend.test")
		t.Setenv("GOOGLE_CLIENT_ID", "client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
		t.Setenv("COOKIE_DOMAIN", "kompozicijas.xyz")
		t.Setenv("COOKIE_SECURE", "false")
		handler, err := newConfiguredAuthHandler(&stubAuthStore{})
		if err != nil {
			t.Fatalf("newConfiguredAuthHandler() error = %v", err)
		}
		if !handler.config.secureCookie {
			t.Fatal("secureCookie = false; want true for https base url")
		}
		if handler.config.frontendOrigin != "https://frontend.test" {
			t.Fatalf("frontendOrigin = %q; want https://frontend.test", handler.config.frontendOrigin)
		}
		if handler.config.cookieDomain != "kompozicijas.xyz" {
			t.Fatalf("cookieDomain = %q; want kompozicijas.xyz", handler.config.cookieDomain)
		}
	})

	t.Run("configured auth handler rejects insecure non-localhost base url", func(t *testing.T) {
		t.Setenv("BASE_URL", "http://backend.test")
		t.Setenv("FRONTEND_URL", "http://frontend.test")
		t.Setenv("GOOGLE_CLIENT_ID", "client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
		t.Setenv("COOKIE_SECURE", "true")

		_, err := newConfiguredAuthHandler(&stubAuthStore{})
		if err == nil || err.Error() != "BASE_URL must use https outside localhost when auth cookies are enabled" {
			t.Fatalf("newConfiguredAuthHandler() error = %v; want https requirement", err)
		}
	})

	t.Run("secure cookie validates base url and localhost override", func(t *testing.T) {
		if _, err := secureCookieFromBaseURL("://bad"); err == nil || err.Error() != "BASE_URL must be a valid absolute URL" {
			t.Fatalf("secureCookieFromBaseURL(invalid) error = %v; want absolute URL error", err)
		}

		t.Setenv("COOKIE_SECURE", "true")
		secure, err := secureCookieFromBaseURL("http://localhost:3000")
		if err != nil {
			t.Fatalf("secureCookieFromBaseURL(localhost) error = %v", err)
		}
		if !secure {
			t.Fatal("secureCookieFromBaseURL(localhost) = false; want true with COOKIE_SECURE=true")
		}

		t.Setenv("COOKIE_SECURE", "false")
		secure, err = secureCookieFromBaseURL("http://127.0.0.1:3000")
		if err != nil {
			t.Fatalf("secureCookieFromBaseURL(127.0.0.1) error = %v", err)
		}
		if secure {
			t.Fatal("secureCookieFromBaseURL(127.0.0.1) = true; want false")
		}
	})

	t.Run("configured auth handler returns intermediate config errors", func(t *testing.T) {
		t.Setenv("GOOGLE_CLIENT_ID", "client-id")
		t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
		t.Setenv("BASE_URL", "frontend.test")
		if _, err := newConfiguredAuthHandler(&stubAuthStore{}); err == nil || err.Error() != "BASE_URL must be a valid absolute URL" {
			t.Fatalf("newConfiguredAuthHandler(base url) error = %v; want BASE_URL error", err)
		}

		t.Setenv("BASE_URL", "https://backend.test")
		t.Setenv("FRONTEND_URL", "frontend.test")
		if _, err := newConfiguredAuthHandler(&stubAuthStore{}); err == nil || err.Error() != "FRONTEND_URL must be a valid absolute URL" {
			t.Fatalf("newConfiguredAuthHandler(frontend url) error = %v; want FRONTEND_URL error", err)
		}

		t.Setenv("FRONTEND_URL", "https://frontend.test")
		t.Setenv("COOKIE_DOMAIN", "https://bad.test")
		if _, err := newConfiguredAuthHandler(&stubAuthStore{}); err == nil || err.Error() != "COOKIE_DOMAIN must be a valid cookie domain" {
			t.Fatalf("newConfiguredAuthHandler(cookie domain) error = %v; want COOKIE_DOMAIN error", err)
		}
	})

	t.Run("frontend origin fallback", func(t *testing.T) {
		handler := &authHandler{config: authConfig{frontendOrigin: "https://configured.test"}}
		if got := frontendOriginFromHandler(handler); got != "https://configured.test" {
			t.Fatalf("frontendOriginFromHandler(handler) = %q; want configured origin", got)
		}

		t.Setenv("FRONTEND_URL", "https://frontend.test")
		if got := frontendOriginFromHandler(nil); got != "https://frontend.test" {
			t.Fatalf("frontendOriginFromHandler(nil) = %q; want https://frontend.test", got)
		}

		t.Setenv("FRONTEND_URL", "frontend.test")
		if got := frontendOriginFromHandler(nil); got != "" {
			t.Fatalf("frontendOriginFromHandler(invalid env) = %q; want empty", got)
		}
	})

	t.Run("random token returns entropy errors", func(t *testing.T) {
		originalReadRandom := readRandom
		defer func() { readRandom = originalReadRandom }()
		readRandom = failingRandomReader{}.Read

		if _, err := randomToken(); err == nil || err.Error() != "random failure" {
			t.Fatalf("randomToken() error = %v; want random failure", err)
		}
	})

	t.Run("frontend config helpers cover defaults", func(t *testing.T) {
		t.Setenv("FRONTEND_URL", "")
		frontendURL, frontendOrigin, err := frontendConfigFromEnv()
		if err != nil {
			t.Fatalf("frontendConfigFromEnv() error = %v", err)
		}
		if frontendURL != "/" {
			t.Fatalf("frontendURL = %q; want /", frontendURL)
		}
		if frontendOrigin != "" {
			t.Fatalf("frontendOrigin = %q; want empty", frontendOrigin)
		}

		t.Setenv("FRONTEND_URL", "frontend.test")
		if _, _, err := frontendConfigFromEnv(); err == nil || err.Error() != "FRONTEND_URL must be a valid absolute URL" {
			t.Fatalf("frontendConfigFromEnv() error = %v; want FRONTEND_URL must be a valid absolute URL", err)
		}

		t.Setenv("FRONTEND_URL", "http://frontend.test/")
		frontendURL, frontendOrigin, err = frontendConfigFromEnv()
		if err != nil {
			t.Fatalf("frontendConfigFromEnv() error = %v", err)
		}
		if frontendURL != "http://frontend.test/" {
			t.Fatalf("frontendURL = %q; want http://frontend.test/", frontendURL)
		}
		if frontendOrigin != "http://frontend.test" {
			t.Fatalf("frontendOrigin = %q; want http://frontend.test", frontendOrigin)
		}

		if domain, err := cookieDomainFromEnv(); err != nil {
			t.Fatalf("cookieDomainFromEnv() error = %v", err)
		} else if domain != "" {
			t.Fatalf("cookieDomain = %q; want empty", domain)
		}

		t.Setenv("COOKIE_DOMAIN", "https://kompozicijas.xyz")
		if _, err := cookieDomainFromEnv(); err == nil || err.Error() != "COOKIE_DOMAIN must be a valid cookie domain" {
			t.Fatalf("cookieDomainFromEnv() error = %v; want invalid cookie domain", err)
		}

		t.Setenv("COOKIE_DOMAIN", ".kompozicijas.xyz")
		if domain, err := cookieDomainFromEnv(); err != nil {
			t.Fatalf("cookieDomainFromEnv() error = %v", err)
		} else if domain != "kompozicijas.xyz" {
			t.Fatalf("cookieDomain = %q; want kompozicijas.xyz", domain)
		}
	})

	t.Run("state digest comparison is strict", func(t *testing.T) {
		if !matchDigest("abc", "abc") {
			t.Fatal("matchDigest(equal) = false; want true")
		}
		if matchDigest("abc", "abcd") {
			t.Fatal("matchDigest(different length) = true; want false")
		}
		if matchDigest("abc", "abd") {
			t.Fatal("matchDigest(different content) = true; want false")
		}
	})

	t.Run("cookie helpers handle nil and blank inputs", func(t *testing.T) {
		if _, err := readAuthCookie(nil); !errors.Is(err, errAuthenticationRequired) {
			t.Fatalf("readAuthCookie(nil) error = %v; want errAuthenticationRequired", err)
		}

		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "   "})
		if _, err := readAuthCookie(request); !errors.Is(err, errAuthenticationRequired) {
			t.Fatalf("readAuthCookie(blank) error = %v; want errAuthenticationRequired", err)
		}

		setCookie(nil, &http.Cookie{Name: authCookieName})
		setCookie(httptest.NewRecorder(), nil)
		clearCookie(httptest.NewRecorder(), nil)
		setNoStore(nil)
	})

	t.Run("cors helpers handle false and true branches", func(t *testing.T) {
		t.Setenv("FRONTEND_URL", "http://frontend.test")

		setCORSHeaders(nil, httptest.NewRequest(http.MethodGet, "/auth/session", nil))
		setCORSHeaders(httptest.NewRecorder(), nil)

		mismatchResponse := httptest.NewRecorder()
		mismatchRequest := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		mismatchRequest.Header.Set("Origin", "http://evil.test")
		setCORSHeaders(mismatchResponse, mismatchRequest)
		if got := mismatchResponse.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("mismatch CORS origin = %q; want empty", got)
		}

		matchResponse := httptest.NewRecorder()
		matchRequest := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
		matchRequest.Header.Set("Origin", "http://frontend.test")
		setCORSHeaders(matchResponse, matchRequest)
		if got := matchResponse.Header().Get("Access-Control-Allow-Origin"); got != "http://frontend.test" {
			t.Fatalf("matched CORS origin = %q; want http://frontend.test", got)
		}

		if handled := handleCORSPreflight(matchResponse, httptest.NewRequest(http.MethodGet, "/auth/session", nil)); handled {
			t.Fatal("handleCORSPreflight(GET) = true; want false")
		}
		preflightResponse := httptest.NewRecorder()
		preflightRequest := httptest.NewRequest(http.MethodOptions, "/auth/session", nil)
		preflightRequest.Header.Set("Origin", "http://frontend.test")
		if handled := handleCORSPreflight(preflightResponse, preflightRequest); !handled {
			t.Fatal("handleCORSPreflight(OPTIONS) = false; want true")
		}
		if preflightResponse.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d; want 204", preflightResponse.Code)
		}
	})
}

func TestSessionFromRequestErrors(t *testing.T) {
	if _, err := (*authHandler)(nil).sessionFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); err == nil || err.Error() != "auth handler is not configured" {
		t.Fatalf("sessionFromRequest(nil handler) error = %v; want auth handler is not configured", err)
	}

	handler := &authHandler{store: &stubAuthStore{sessionErr: errors.New("session boom")}, now: time.Now}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	if _, err := handler.sessionFromRequest(request); err == nil || err.Error() != "session boom" {
		t.Fatalf("sessionFromRequest(store error) error = %v; want session boom", err)
	}

	handler = &authHandler{store: &stubAuthStore{sessionUser: database.SessionUserRecord{}}, now: time.Now}
	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	if _, err := handler.sessionFromRequest(request); !errors.Is(err, errAuthenticationRequired) {
		t.Fatalf("sessionFromRequest(empty user) error = %v; want errAuthenticationRequired", err)
	}
}

func TestHandleSessionAndLogoutErrors(t *testing.T) {
	t.Setenv("FRONTEND_URL", "")
	handler := &authHandler{store: &stubAuthStore{sessionErr: errors.New("session boom")}, now: time.Now}
	request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	response := httptest.NewRecorder()
	handler.handleSession(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("handleSession() status = %d; want 500", response.Code)
	}

	handler = &authHandler{store: &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "user-1", Email: "player@example.com"}}, now: time.Now}
	request = httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	handler.handleSession(failingResponseWriter{}, request)

	handler = &authHandler{store: &stubAuthStore{deleteErr: errors.New("delete boom")}, now: time.Now}
	request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	response = httptest.NewRecorder()
	handler.handleLogout(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("handleLogout() status = %d; want 204", response.Code)
	}
}

func TestFetchGoogleUserCoverage(t *testing.T) {
	token := &oauth2.Token{AccessToken: "access-token", TokenType: "Bearer"}
	handler := &authHandler{config: authConfig{oauthConfig: &oauth2.Config{}}}

	t.Run("request failure", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, newOAuthHTTPClient(nil, errors.New("userinfo boom"), http.StatusOK, `{"access_token":"token","token_type":"Bearer"}`, ``))
		if _, err := handler.fetchGoogleUser(ctx, token); err == nil || !strings.Contains(err.Error(), "userinfo boom") {
			t.Fatalf("fetchGoogleUser() error = %v; want wrapped userinfo boom", err)
		}
	})

	t.Run("unexpected status", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonHTTPResponse(http.StatusUnauthorized, `{}`), nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		if _, err := handler.fetchGoogleUser(ctx, token); err == nil || !strings.Contains(err.Error(), "unexpected status") {
			t.Fatalf("fetchGoogleUser() error = %v; want unexpected status", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonHTTPResponse(http.StatusOK, `{`), nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		if _, err := handler.fetchGoogleUser(ctx, token); err == nil {
			t.Fatal("fetchGoogleUser() error = nil; want json decode error")
		}
	})

	t.Run("missing subject", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonHTTPResponse(http.StatusOK, `{"email":"player@example.com","email_verified":true}`), nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		if _, err := handler.fetchGoogleUser(ctx, token); err == nil || err.Error() != "google user info missing subject" {
			t.Fatalf("fetchGoogleUser() error = %v; want missing subject", err)
		}
	})

	t.Run("requires verified email", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonHTTPResponse(http.StatusOK, `{"sub":"123","email":"player@example.com","email_verified":false}`), nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		if _, err := handler.fetchGoogleUser(ctx, token); err == nil || err.Error() != "google account email is not verified" {
			t.Fatalf("fetchGoogleUser() error = %v; want email not verified", err)
		}
	})

	t.Run("falls back to email for name", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return jsonHTTPResponse(http.StatusOK, `{"sub":"123","email":"Player@Example.com","email_verified":true,"picture":" https://cdn.example.com/player.png "}`), nil
		})}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
		user, err := handler.fetchGoogleUser(ctx, token)
		if err != nil {
			t.Fatalf("fetchGoogleUser() error = %v", err)
		}
		if user.ID != "" || user.Provider != googleProvider || user.ProviderAccountID != "123" || user.Name != "Player@Example.com" || user.Email != "player@example.com" || user.Image != "https://cdn.example.com/player.png" {
			t.Fatalf("user = %#v; want normalized google user", user)
		}
	})
}

func TestHandleGoogleSignInAndCallbackCoverage(t *testing.T) {
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	t.Setenv("FRONTEND_URL", "http://frontend.test")

	t.Run("sign in handles state and random errors", func(t *testing.T) {
		handler := &authHandler{
			config: authConfig{oauthConfig: &oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "https://oauth.test/auth"}}},
			now:    func() time.Time { return now },
			state:  func() (string, error) { return "", errors.New("state boom") },
		}
		response := httptest.NewRecorder()
		handler.handleGoogleSignIn(response, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("handleGoogleSignIn(state error) status = %d; want 500", response.Code)
		}

		originalReadRandom := readRandom
		defer func() { readRandom = originalReadRandom }()
		readRandom = failingRandomReader{}.Read
		handler = &authHandler{
			config: authConfig{oauthConfig: &oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "https://oauth.test/auth"}}},
			now:    func() time.Time { return now },
			state:  func() (string, error) { return "state-token", nil },
		}
		response = httptest.NewRecorder()
		handler.handleGoogleSignIn(response, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("handleGoogleSignIn(random error) status = %d; want 500", response.Code)
		}
	})

	t.Run("callback covers validation and persistence branches", func(t *testing.T) {
		t.Run("invalid form", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?%zz", nil)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; want 400", response.Code)
			}
		})

		t.Run("oauth provider error", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?error=access_denied", nil)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", response.Code)
			}
		})

		t.Run("missing params", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=state-token", nil)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; want 400", response.Code)
			}
		})

		t.Run("invalid state", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=state-token&code=code", nil)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", response.Code)
			}
		})

		t.Run("missing pkce", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=state-token&code=code", nil)
			request.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: stateDigest("state-token")})
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", response.Code)
			}
		})

		t.Run("exchange failure", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			request := newOAuthCallbackRequest("state=state-token&code=code", newOAuthHTTPClient(errors.New("exchange boom"), nil, http.StatusOK, ``, ``))
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", response.Code)
			}
		})

		t.Run("user fetch failure", func(t *testing.T) {
			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":""}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d; want 401", response.Code)
			}
		})

		t.Run("upsert failure", func(t *testing.T) {
			store := &stubAuthStore{upsertErr: errors.New("upsert boom")}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d; want 500", response.Code)
			}
		})

		t.Run("session token generation failure", func(t *testing.T) {
			originalReadRandom := readRandom
			defer func() { readRandom = originalReadRandom }()
			readRandom = failingRandomReader{}.Read

			handler := newOAuthCallbackHandler(&stubAuthStore{}, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d; want 500", response.Code)
			}
		})

		t.Run("create session failure", func(t *testing.T) {
			store := &stubAuthStore{createErr: errors.New("create boom")}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d; want 500", response.Code)
			}
		})

		t.Run("user conflict returns conflict status", func(t *testing.T) {
			store := &stubAuthStore{upsertErr: database.ErrUserConflict}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d; want 409", response.Code)
			}
		})

		t.Run("success", func(t *testing.T) {
			store := &stubAuthStore{upsertedUser: authenticatedUser{ID: "user-123", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png", Provider: googleProvider, ProviderAccountID: "123"}}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true,"picture":"https://cdn.example.com/player.png"}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusFound {
				t.Fatalf("status = %d; want 302", response.Code)
			}
			if got := response.Header().Get("Location"); got != "http://frontend.test/" {
				t.Fatalf("redirect location = %q; want http://frontend.test/", got)
			}
			if len(store.upsertedUsers) != 1 || len(store.createdSessions) != 1 {
				t.Fatalf("persisted records = users:%d sessions:%d; want 1 each", len(store.upsertedUsers), len(store.createdSessions))
			}
			if store.createdSessions[0].UserID != "user-123" {
				t.Fatalf("session user id = %q; want user-123", store.createdSessions[0].UserID)
			}
			cookies := response.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("cookies = 0; want session cookie")
			}
			if cookies[0].Domain != "" {
				t.Fatalf("session cookie domain = %q; want empty by default", cookies[0].Domain)
			}
		})

		t.Run("success reuses existing same user session", func(t *testing.T) {
			store := &stubAuthStore{
				upsertedUser: authenticatedUser{ID: "user-123", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png", Provider: googleProvider, ProviderAccountID: "123"},
				sessionUser: database.SessionUserRecord{
					ID:        "user-123",
					Name:      "Player One",
					Email:     "player@example.com",
					ImageURL:  "https://cdn.example.com/player.png",
					ExpiresAt: now.Add(time.Hour),
				},
			}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true,"picture":"https://cdn.example.com/player.png"}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			request.AddCookie(&http.Cookie{Name: authCookieName, Value: "existing-session-token"})
			response := httptest.NewRecorder()

			handler.handleGoogleCallback(response, request)

			if response.Code != http.StatusFound {
				t.Fatalf("status = %d; want 302", response.Code)
			}
			if len(store.createdSessions) != 1 {
				t.Fatalf("created sessions = %d; want 1 renewal", len(store.createdSessions))
			}
			if store.createdSessions[0].Token != "existing-session-token" {
				t.Fatalf("renewed session token = %q; want existing-session-token", store.createdSessions[0].Token)
			}
			if len(store.deletedTokens) != 0 {
				t.Fatalf("deleted tokens = %#v; want none", store.deletedTokens)
			}
			if got := cookieValue(response.Result().Cookies(), authCookieName); got != "existing-session-token" {
				t.Fatalf("session cookie = %q; want existing-session-token", got)
			}
		})

		t.Run("success replaces existing different user session", func(t *testing.T) {
			store := &stubAuthStore{
				upsertedUser: authenticatedUser{ID: "user-123", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png", Provider: googleProvider, ProviderAccountID: "123"},
				sessionUser: database.SessionUserRecord{
					ID:        "other-user",
					Name:      "Other Player",
					Email:     "other@example.com",
					ExpiresAt: now.Add(time.Hour),
				},
			}
			handler := newOAuthCallbackHandler(store, now)
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true,"picture":"https://cdn.example.com/player.png"}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			request.AddCookie(&http.Cookie{Name: authCookieName, Value: "other-session-token"})
			response := httptest.NewRecorder()

			handler.handleGoogleCallback(response, request)

			if response.Code != http.StatusFound {
				t.Fatalf("status = %d; want 302", response.Code)
			}
			if len(store.deletedTokens) != 1 || store.deletedTokens[0] != "other-session-token" {
				t.Fatalf("deleted tokens = %#v; want other-session-token", store.deletedTokens)
			}
			if len(store.createdSessions) != 1 {
				t.Fatalf("created sessions = %d; want 1", len(store.createdSessions))
			}
			if store.createdSessions[0].Token == "" || store.createdSessions[0].Token == "other-session-token" {
				t.Fatalf("created session token = %q; want a new token", store.createdSessions[0].Token)
			}
		})

		t.Run("success with shared cookie domain", func(t *testing.T) {
			store := &stubAuthStore{upsertedUser: authenticatedUser{ID: "user-123", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png", Provider: googleProvider, ProviderAccountID: "123"}}
			handler := newOAuthCallbackHandler(store, now)
			handler.config.cookieDomain = "kompozicijas.xyz"
			client := newOAuthHTTPClient(nil, nil, http.StatusOK, `{"access_token":"access-token","token_type":"Bearer"}`, `{"sub":"123","name":"Player One","email":"player@example.com","email_verified":true,"picture":"https://cdn.example.com/player.png"}`)
			request := newOAuthCallbackRequest("state=state-token&code=code", client)
			response := httptest.NewRecorder()
			handler.handleGoogleCallback(response, request)
			if response.Code != http.StatusFound {
				t.Fatalf("status = %d; want 302", response.Code)
			}
			cookies := response.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("cookies = 0; want session cookie")
			}
			if cookies[0].Domain != "kompozicijas.xyz" {
				t.Fatalf("session cookie domain = %q; want kompozicijas.xyz", cookies[0].Domain)
			}
		})
	})
}

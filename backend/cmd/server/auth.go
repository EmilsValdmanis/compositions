package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

var readRandom = rand.Read

const (
	authCookieName         = "compositions_session"
	oauthStateCookieName   = "compositions_oauth_state"
	oauthPKCECookieName    = "compositions_oauth_pkce"
	oauthStateCookieMaxAge = 10 * time.Minute
	defaultSessionLifetime = 30 * 24 * time.Hour
	authCookieSameSite     = http.SameSiteLaxMode
	googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"
	googleProvider         = "google"
)

type authSession struct {
	user  authenticatedUser
	token string
	valid bool
}

type sessionReader interface {
	GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (database.SessionUserRecord, error)
}

type sessionWriter interface {
	UpsertUser(ctx context.Context, user authenticatedUser) (authenticatedUser, error)
	CreateSession(ctx context.Context, session authSessionRecord) error
	DeleteSession(ctx context.Context, sessionToken string) error
}

type authSessionRecord struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type authStore interface {
	sessionReader
	sessionWriter
	Close() error
}

type authConfig struct {
	baseURL        string
	frontendURL    string
	frontendOrigin string
	cookieDomain   string
	secureCookie   bool
	oauthConfig    *oauth2.Config
	sessionTTL     time.Duration
}

type authHandler struct {
	config authConfig
	store  authStore
	now    func() time.Time
	state  func() (string, error)
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
}

type sessionResponse struct {
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Image string `json:"image"`
	} `json:"user"`
}

func baseURLFromEnv() (string, error) {
	return absoluteURLFromEnv("BASE_URL")
}

func absoluteURLFromEnv(envName string) (string, error) {
	return absoluteURLFromString(os.Getenv(envName), envName)
}

func absoluteURLFromString(rawURL, envName string) (string, error) {
	absoluteURL := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if absoluteURL == "" {
		return "", fmt.Errorf("%s is required", envName)
	}
	parsed, err := url.Parse(absoluteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%s must be a valid absolute URL", envName)
	}
	return absoluteURL, nil
}

func frontendConfigFromEnv() (string, string, error) {
	frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if frontendURL == "" {
		return "/", "", nil
	}

	absoluteURL, err := absoluteURLFromString(frontendURL, "FRONTEND_URL")
	if err != nil {
		return "", "", err
	}
	parsed, _ := url.Parse(absoluteURL)
	return absoluteURL + "/", parsed.Scheme + "://" + parsed.Host, nil
}

func cookieDomainFromEnv() (string, error) {
	rawDomain := strings.TrimSpace(os.Getenv("COOKIE_DOMAIN"))
	if rawDomain == "" {
		return "", nil
	}

	domain := strings.TrimPrefix(rawDomain, ".")
	if domain == "" || strings.Contains(domain, "://") || strings.Contains(domain, "/") || strings.Contains(domain, ":") {
		return "", errors.New("COOKIE_DOMAIN must be a valid cookie domain")
	}

	return domain, nil
}

func secureCookieFromBaseURL(baseURL string) (bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return false, errors.New("BASE_URL must be a valid absolute URL")
	}
	if parsed.Scheme == "https" {
		return true, nil
	}
	if isLocalhostHost(parsed.Hostname()) {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true") {
			return true, nil
		}
		return false, nil
	}
	return false, errors.New("BASE_URL must use https outside localhost when auth cookies are enabled")
}

func isLocalhostHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func newGoogleOAuthConfigFromEnv() (*oauth2.Config, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID is required")
	}
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	if clientSecret == "" {
		return nil, errors.New("GOOGLE_CLIENT_SECRET is required")
	}
	baseURL, err := baseURLFromEnv()
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     googleoauth.Endpoint,
		RedirectURL:  baseURL + "/auth/google/callback",
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}, nil
}

func newConfiguredAuthHandler(store authStore) (*authHandler, error) {
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	oauthConfig, err := newGoogleOAuthConfigFromEnv()
	if err != nil {
		return nil, err
	}
	baseURL, _ := baseURLFromEnv()
	frontendURL, frontendOrigin, err := frontendConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cookieDomain, err := cookieDomainFromEnv()
	if err != nil {
		return nil, err
	}
	secureCookie, err := secureCookieFromBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	return &authHandler{
		config: authConfig{
			baseURL:        baseURL,
			frontendURL:    frontendURL,
			frontendOrigin: frontendOrigin,
			cookieDomain:   cookieDomain,
			secureCookie:   secureCookie,
			oauthConfig:    oauthConfig,
			sessionTTL:     defaultSessionLifetime,
		},
		store: store,
		now:   time.Now,
		state: randomToken,
	}, nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := readRandom(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func stateDigest(state string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(state)))
	return hex.EncodeToString(hash[:])
}

func (h *authHandler) handleGoogleSignIn(w http.ResponseWriter, r *http.Request) {
	state, err := h.state()
	if err != nil {
		slog.Error("generate oauth state failed", "error", err)
		http.Error(w, "failed to start oauth flow", http.StatusInternalServerError)
		return
	}
	verifier, err := randomToken()
	if err != nil {
		slog.Error("generate oauth pkce verifier failed", "error", err)
		http.Error(w, "failed to start oauth flow", http.StatusInternalServerError)
		return
	}

	setCookie(w, h.cookie(oauthStateCookieName, stateDigest(state), h.now().Add(oauthStateCookieMaxAge)))
	setCookie(w, h.cookie(oauthPKCECookieName, verifier, h.now().Add(oauthStateCookieMaxAge)))
	http.Redirect(w, r, h.config.oauthConfig.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
	), http.StatusFound)
}

func (h *authHandler) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid oauth callback", http.StatusBadRequest)
		return
	}
	if oauthError := strings.TrimSpace(r.FormValue("error")); oauthError != "" {
		http.Error(w, "google oauth failed", http.StatusUnauthorized)
		return
	}
	state := strings.TrimSpace(r.FormValue("state"))
	code := strings.TrimSpace(r.FormValue("code"))
	if state == "" || code == "" {
		http.Error(w, "missing oauth callback parameters", http.StatusBadRequest)
		return
	}
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || !matchDigest(stateCookie.Value, stateDigest(state)) {
		http.Error(w, "invalid oauth state", http.StatusUnauthorized)
		return
	}
	pkceCookie, err := r.Cookie(oauthPKCECookieName)
	if err != nil || strings.TrimSpace(pkceCookie.Value) == "" {
		http.Error(w, "missing oauth pkce verifier", http.StatusUnauthorized)
		return
	}
	clearCookie(w, h.cookie(oauthStateCookieName, "", time.Unix(0, 0)))
	clearCookie(w, h.cookie(oauthPKCECookieName, "", time.Unix(0, 0)))

	token, err := h.config.oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(pkceCookie.Value))
	if err != nil {
		slog.Warn("oauth code exchange failed", "error", err)
		http.Error(w, "google oauth exchange failed", http.StatusUnauthorized)
		return
	}
	user, err := h.fetchGoogleUser(r.Context(), token)
	if err != nil {
		slog.Warn("fetch google user failed", "error", err)
		http.Error(w, "failed to read google profile", http.StatusUnauthorized)
		return
	}
	user, err = h.store.UpsertUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, database.ErrUserConflict) {
			slog.Warn("save oauth user conflict", "provider", user.Provider, "providerAccountID", user.ProviderAccountID, "email", user.Email)
			http.Error(w, "google account could not be linked", http.StatusConflict)
			return
		}
		slog.Error("save oauth user failed", "provider", user.Provider, "providerAccountID", user.ProviderAccountID, "error", err)
		http.Error(w, "failed to save session user", http.StatusInternalServerError)
		return
	}

	expiresAt := h.now().Add(h.config.sessionTTL).UTC()
	sessionToken, err := h.sessionTokenForUser(r, user, expiresAt)
	if err != nil {
		slog.Error("persist session failed", "userID", user.ID, "error", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	setCookie(w, h.cookie(authCookieName, sessionToken, expiresAt))
	http.Redirect(w, r, h.config.frontendURL, http.StatusFound)
}

func (h *authHandler) sessionTokenForUser(r *http.Request, user authenticatedUser, expiresAt time.Time) (string, error) {
	session, err := h.sessionFromRequest(r)
	if err == nil {
		if session.user.ID == user.ID {
			return session.token, h.store.CreateSession(r.Context(), authSessionRecord{
				Token:     session.token,
				UserID:    user.ID,
				ExpiresAt: expiresAt,
			})
		}
		if deleteErr := h.store.DeleteSession(r.Context(), session.token); deleteErr != nil && !errors.Is(deleteErr, database.ErrSessionNotFound) {
			slog.Warn("delete previous user session failed", "error", deleteErr)
		}
	} else if !errors.Is(err, errAuthenticationRequired) {
		return "", err
	}

	sessionToken, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := h.store.CreateSession(r.Context(), authSessionRecord{Token: sessionToken, UserID: user.ID, ExpiresAt: expiresAt}); err != nil {
		return "", err
	}
	return sessionToken, nil
}

func (h *authHandler) handleSession(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	session, err := h.sessionFromRequest(r)
	if err != nil {
		if errors.Is(err, errAuthenticationRequired) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("null"))
			return
		}
		slog.Error("resolve session failed", "error", err)
		http.Error(w, "failed to load session", http.StatusInternalServerError)
		return
	}

	var payload sessionResponse
	payload.User.ID = session.user.ID
	payload.User.Name = session.user.Name
	payload.User.Email = session.user.Email
	payload.User.Image = session.user.Image
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write session response failed", "error", err)
	}
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionToken, _ := readAuthCookie(r)
	if sessionToken != "" {
		if err := h.store.DeleteSession(r.Context(), sessionToken); err != nil && !errors.Is(err, database.ErrSessionNotFound) && !errors.Is(err, errAuthenticationRequired) {
			slog.Warn("delete session failed", "error", err)
		}
	}
	clearCookie(w, h.cookie(authCookieName, "", time.Unix(0, 0)))
	clearCookie(w, h.cookie(oauthStateCookieName, "", time.Unix(0, 0)))
	clearCookie(w, h.cookie(oauthPKCECookieName, "", time.Unix(0, 0)))
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) sessionFromRequest(r *http.Request) (authSession, error) {
	if h == nil || h.store == nil {
		return authSession{}, errors.New("auth handler is not configured")
	}
	sessionToken, err := readAuthCookie(r)
	if err != nil {
		return authSession{}, errAuthenticationRequired
	}
	record, err := h.store.GetSessionUserByToken(r.Context(), sessionToken, h.now())
	if err != nil {
		if errors.Is(err, database.ErrSessionNotFound) {
			return authSession{}, errAuthenticationRequired
		}
		return authSession{}, err
	}

	user := authenticatedUser{
		ID:    record.ID,
		Name:  record.Name,
		Email: record.Email,
		Image: record.ImageURL,
	}
	if !user.isAuthenticated() {
		return authSession{}, errAuthenticationRequired
	}

	return authSession{user: user, token: sessionToken, valid: true}, nil
}

func (h *authHandler) fetchGoogleUser(ctx context.Context, token *oauth2.Token) (authenticatedUser, error) {
	client := h.config.oauthConfig.Client(ctx, token)
	resp, err := client.Get(googleUserInfoEndpoint)
	if err != nil {
		return authenticatedUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return authenticatedUser{}, fmt.Errorf("google user info: unexpected status %s", resp.Status)
	}

	var profile googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return authenticatedUser{}, err
	}
	if strings.TrimSpace(profile.Sub) == "" {
		return authenticatedUser{}, errors.New("google user info missing subject")
	}
	if !profile.EmailVerified || strings.TrimSpace(profile.Email) == "" {
		return authenticatedUser{}, errors.New("google account email is not verified")
	}

	name := strings.TrimSpace(profile.Name)
	if name == "" {
		name = strings.TrimSpace(profile.Email)
	}

	return authenticatedUser{
		Name:              name,
		Email:             strings.ToLower(strings.TrimSpace(profile.Email)),
		Image:             strings.TrimSpace(profile.Picture),
		Provider:          googleProvider,
		ProviderAccountID: strings.TrimSpace(profile.Sub),
	}, nil
}

func readAuthCookie(r *http.Request) (string, error) {
	if r == nil {
		return "", errAuthenticationRequired
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", errAuthenticationRequired
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", errAuthenticationRequired
	}
	return value, nil
}

func (h *authHandler) cookie(name, value string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   h.config.cookieDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.config.secureCookie,
		SameSite: authCookieSameSite,
		Expires:  expiresAt,
	}
}

func setCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if w == nil || cookie == nil {
		return
	}
	http.SetCookie(w, cookie)
}

func clearCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	setCookie(w, cookie)
}

func setNoStore(w http.ResponseWriter) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if origin := frontendOriginFromHandler(nil); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
	}
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil {
		return
	}
	origin := frontendOriginFromHandler(nil)
	if origin == "" || r.Header.Get("Origin") != origin {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleCORSPreflight(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.Method != http.MethodOptions {
		return false
	}
	setCORSHeaders(w, r)
	w.WriteHeader(http.StatusNoContent)
	return true
}

func matchDigest(left, right string) bool {
	leftBytes := []byte(strings.TrimSpace(left))
	rightBytes := []byte(strings.TrimSpace(right))
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}

func frontendOriginFromHandler(h *authHandler) string {
	if h != nil {
		return h.config.frontendOrigin
	}
	_, frontendOrigin, err := frontendConfigFromEnv()
	if err != nil {
		return ""
	}
	return frontendOrigin
}

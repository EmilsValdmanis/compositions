package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var errAuthenticationRequired = errors.New("authentication required")

type authenticatedUser struct {
	ID    string
	Name  string
	Email string
	Image string
}

func (u authenticatedUser) isAuthenticated() bool {
	return strings.TrimSpace(u.ID) != ""
}

func (u authenticatedUser) displayName() string {
	if name := strings.TrimSpace(u.Name); name != "" {
		return name
	}
	if email := strings.TrimSpace(u.Email); email != "" {
		return email
	}
	return strings.TrimSpace(u.ID)
}

type sessionVerifier interface {
	VerifySession(ctx context.Context, bearerToken string) (authenticatedUser, error)
}

type betterAuthSessionVerifier struct {
	baseURL string
	client  *http.Client
}

type betterAuthSessionResponse struct {
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Image string `json:"image"`
	} `json:"user"`
}

func newBetterAuthSessionVerifierFromEnv() (sessionVerifier, error) {
	baseURL, err := betterAuthBaseURLFromEnv()
	if err != nil {
		return nil, err
	}
	return newBetterAuthSessionVerifier(baseURL, nil), nil
}

func betterAuthBaseURLFromEnv() (string, error) {
	baseURL := strings.TrimSpace(os.Getenv("BETTER_AUTH_URL"))
	if baseURL == "" {
		return "", errors.New("BETTER_AUTH_URL is required")
	}
	return baseURL, nil
}

func newBetterAuthSessionVerifier(baseURL string, client *http.Client) sessionVerifier {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	return &betterAuthSessionVerifier{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (v *betterAuthSessionVerifier) VerifySession(ctx context.Context, bearerToken string) (authenticatedUser, error) {
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return authenticatedUser{}, errAuthenticationRequired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/api/auth/get-session", nil)
	if err != nil {
		return authenticatedUser{}, fmt.Errorf("build auth session request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.client.Do(req)
	if err != nil {
		return authenticatedUser{}, fmt.Errorf("verify auth session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return authenticatedUser{}, errAuthenticationRequired
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return authenticatedUser{}, fmt.Errorf("verify auth session: unexpected status %s", resp.Status)
	}

	var payload *betterAuthSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return authenticatedUser{}, fmt.Errorf("decode auth session response: %w", err)
	}
	if payload == nil {
		return authenticatedUser{}, errAuthenticationRequired
	}

	user := authenticatedUser{
		ID:    strings.TrimSpace(payload.User.ID),
		Name:  strings.TrimSpace(payload.User.Name),
		Email: strings.TrimSpace(payload.User.Email),
		Image: strings.TrimSpace(payload.User.Image),
	}
	if !user.isAuthenticated() || user.displayName() == "" {
		return authenticatedUser{}, errAuthenticationRequired
	}

	return user, nil
}

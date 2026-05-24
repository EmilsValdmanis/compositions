package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

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

func TestBetterAuthSessionVerifierVerifySession(t *testing.T) {
		t.Run("requires bearer token", func(t *testing.T) {
			verifier := &betterAuthSessionVerifier{}

			user, err := verifier.VerifySession(context.Background(), "   ")
			if !errors.Is(err, errAuthenticationRequired) {
				t.Fatalf("VerifySession() error = %v; want errAuthenticationRequired", err)
			}
		if user != (authenticatedUser{}) {
			t.Fatalf("user = %#v; want zero value", user)
		}
	})

		t.Run("returns request build error for invalid base url", func(t *testing.T) {
			verifier := &betterAuthSessionVerifier{baseURL: "://bad", client: &http.Client{}}

			_, err := verifier.VerifySession(context.Background(), "token")
			if err == nil || err.Error() == "" {
				t.Fatal("VerifySession() error = nil; want build request error")
			}
	})

	t.Run("returns transport error", func(t *testing.T) {
		verifier := &betterAuthSessionVerifier{
			baseURL: "http://frontend.test",
			client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
					t.Fatalf("Authorization header = %q; want Bearer token-123", got)
				}
				return nil, errors.New("network boom")
			})},
		}

			_, err := verifier.VerifySession(context.Background(), " token-123 ")
			if err == nil || !strings.Contains(err.Error(), "network boom") {
				t.Fatalf("VerifySession() error = %v; want wrapped network boom", err)
			}
		})

		t.Run("does not forward cookie header", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
					t.Fatalf("Authorization header = %q; want Bearer token-123", got)
				}
				if got := r.Header.Get("Cookie"); got != "" {
					t.Fatalf("Cookie header = %q; want empty", got)
				}

				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, `{"user":{"id":"user-1","name":"Player One","email":"player@example.com"}}`)
			}))
			defer server.Close()

			verifier := newBetterAuthSessionVerifier(server.URL+"/", server.Client())
			user, err := verifier.VerifySession(context.Background(), "token-123")
			if err != nil {
				t.Fatalf("VerifySession() error = %v", err)
			}
		if user.ID != "user-1" {
			t.Fatalf("user.ID = %q; want user-1", user.ID)
		}
	})

	testCases := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
		wantUser   authenticatedUser
	}{
		{
			name:       "unauthorized response",
			statusCode: http.StatusUnauthorized,
			body:       `{}`,
			wantErr:    errAuthenticationRequired.Error(),
		},
		{
			name:       "unexpected status response",
			statusCode: http.StatusBadGateway,
			body:       `{}`,
			wantErr:    "verify auth session: unexpected status 502 Bad Gateway",
		},
		{
			name:       "invalid json body",
			statusCode: http.StatusOK,
			body:       `{`,
			wantErr:    "decode auth session response: unexpected EOF",
		},
		{
			name:       "null payload",
			statusCode: http.StatusOK,
			body:       `null`,
			wantErr:    errAuthenticationRequired.Error(),
		},
		{
			name:       "missing identity fields",
			statusCode: http.StatusOK,
			body:       `{"user":{"id":"","name":"","email":""}}`,
			wantErr:    errAuthenticationRequired.Error(),
		},
		{
			name:       "success with email fallback",
			statusCode: http.StatusOK,
			body:       `{"user":{"id":"user-1","name":"","email":"player@example.com","image":"https://cdn.example.com/player.png"}}`,
			wantUser: authenticatedUser{
				ID:    "user-1",
				Name:  "",
				Email: "player@example.com",
				Image: "https://cdn.example.com/player.png",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("request method = %s; want GET", r.Method)
				}
				if r.URL.Path != "/api/auth/get-session" {
					t.Fatalf("request path = %q; want /api/auth/get-session", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
					t.Fatalf("Authorization header = %q; want Bearer token-123", got)
				}

				w.WriteHeader(testCase.statusCode)
				_, _ = io.WriteString(w, testCase.body)
			}))
			defer server.Close()

			verifier := newBetterAuthSessionVerifier(server.URL+"/", server.Client())
			user, err := verifier.VerifySession(context.Background(), "token-123")

			if testCase.wantErr != "" {
				if err == nil || err.Error() != testCase.wantErr {
					t.Fatalf("VerifySession() error = %v; want %q", err, testCase.wantErr)
				}
				if user != (authenticatedUser{}) {
					t.Fatalf("user = %#v; want zero value", user)
				}
				return
			}

			if err != nil {
				t.Fatalf("VerifySession() error = %v", err)
			}
			if user != testCase.wantUser {
				t.Fatalf("user = %#v; want %#v", user, testCase.wantUser)
			}
		})
	}
}

package main

import "strings"

var errAuthenticationRequired = errorString("authentication required")

type authenticatedUser struct {
	ID                string
	Name              string
	Email             string
	Image             string
	Provider          string
	ProviderAccountID string
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

type errorString string

func (e errorString) Error() string { return string(e) }

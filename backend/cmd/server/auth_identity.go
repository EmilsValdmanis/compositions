package main

import (
	"strings"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

var errAuthenticationRequired = clientError{"auth_required", "authentication required"}

type authenticatedUser struct {
	ID                string
	Name              string
	Email             string
	Image             string
	IsAdmin           bool
	OnboardingVersion int
	Provider          string
	ProviderAccountID string
}

func (u authenticatedUser) record() database.UserRecord {
	return database.UserRecord{
		ID: u.ID, Name: u.Name, Email: u.Email, ImageURL: u.Image,
		Provider: u.Provider, ProviderAccountID: u.ProviderAccountID,
	}
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

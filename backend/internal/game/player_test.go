package game

import "testing"

func TestNewPlayerInitializesDefaults(t *testing.T) {
	player := NewPlayer()

	if player.ID == "" {
		t.Error("player ID is empty")
	}
	if player.hand == nil {
		t.Error("player hand is nil")
	}
	if player.totalPoints != 0 {
		t.Errorf("player totalPoints = %d; want 0", player.totalPoints)
	}
}

func TestNewPlayerIDsAreUnique(t *testing.T) {
	first := NewPlayer()
	second := NewPlayer()

	if first.ID == second.ID {
		t.Error("expected unique player IDs")
	}
}

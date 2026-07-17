package game

import "testing"

func TestCIProtectionRejectsFailingChecks(t *testing.T) {
	t.Fatal("deliberate failure used to verify main branch protection")
}

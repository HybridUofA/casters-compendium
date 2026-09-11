package cardupdate

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

func TestCarryForwardLegacyIDsMatchesReplacementArtwork(t *testing.T) {
	previous := []gamecards.Card{{
		ID: "235", LegacyIDs: []string{"older"}, Name: "Carella", Type: "Caster",
		CostLevel: "2", CardNumber: "EX02-009", ImageURL: "https://cards/Carella-Lv2.png",
	}}
	current := []gamecards.Card{
		{
			ID: "1208", Name: "Carella", Type: "Caster", CostLevel: "2",
			CardNumber: "EX02-009", ImageURL: "https://cards/Carella-lv2-Rare.png",
		},
		{
			ID: "1225", Name: "Carella", Type: "Caster", CostLevel: "2",
			CardNumber: "EX02-009", ImageURL: "https://cards/Carella-lv2.png",
		},
	}

	updated, migrations := CarryForwardLegacyIDs(previous, current)
	if len(migrations) != 1 || migrations[0].LegacyID != "235" || migrations[0].CanonicalID != "1225" {
		t.Fatalf("migrations = %#v, want 235 -> 1225", migrations)
	}
	if len(updated[1].LegacyIDs) != 2 || updated[1].LegacyIDs[0] != "235" || updated[1].LegacyIDs[1] != "older" {
		t.Fatalf("canonical legacy IDs = %#v, want 235 and older", updated[1].LegacyIDs)
	}
}

func TestCarryForwardLegacyIDsPreservesExistingAliasChain(t *testing.T) {
	previous := []gamecards.Card{{
		ID: "1225", LegacyIDs: []string{"235"}, Name: "Carella", Type: "Caster",
		CostLevel: "2", ImageURL: "https://cards/carella.png",
	}}
	current := []gamecards.Card{{
		ID: "1400", Name: "Carella", Type: "Caster", CostLevel: "2",
		ImageURL: "https://cards/carella.png",
	}}

	updated, _ := CarryForwardLegacyIDs(previous, current)
	want := []string{"1225", "235"}
	if len(updated[0].LegacyIDs) != len(want) {
		t.Fatalf("legacy chain = %#v, want %#v", updated[0].LegacyIDs, want)
	}
	for index := range want {
		if updated[0].LegacyIDs[index] != want[index] {
			t.Fatalf("legacy chain = %#v, want %#v", updated[0].LegacyIDs, want)
		}
	}
}

func TestCarryForwardLegacyIDsDoesNotGuessUnmatchedRemoval(t *testing.T) {
	previous := []gamecards.Card{{ID: "old", Name: "Removed", Type: "Servant"}}
	current := []gamecards.Card{{ID: "new", Name: "Different", Type: "Servant"}}

	updated, migrations := CarryForwardLegacyIDs(previous, current)
	if len(migrations) != 0 || len(updated[0].LegacyIDs) != 0 {
		t.Fatalf("unmatched removal produced aliases: %#v, %#v", updated, migrations)
	}
}

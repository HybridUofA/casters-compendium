package engine

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestManualCorruptOrbMovesToOwnerHand(t *testing.T) {
	state := battleStateForTest()
	revision := state.Revision
	catalog := casterAetherCatalogForTest{}

	if err := ManualCorruptOrb(&state, catalog, "player-one", "p2-orb-2", revision); err != nil {
		t.Fatalf("ManualCorruptOrb() error = %v", err)
	}
	if len(state.Players[1].Orbs) != 1 || state.Players[1].Orbs[0] != "p2-orb-1" {
		t.Fatalf("Orbs = %#v; want p2-orb-1 remaining", state.Players[1].Orbs)
	}
	if len(state.Players[1].Hand) != 1 || state.Players[1].Hand[0] != "p2-orb-2" {
		t.Fatalf("Hand = %#v; want p2-orb-2", state.Players[1].Hand)
	}
	if state.Revision != revision+1 {
		t.Fatalf("Revision = %d; want %d", state.Revision, revision+1)
	}
}

func TestManualBanishOrbMovesToGraveyard(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p1-orb-1"] = battleCard("p1-orb-1", "player-one", model.OrientationRecovered)
	state.Players[0].Orbs = []model.MatchCardID{"p1-orb-1"}
	revision := state.Revision

	if err := ManualBanishOrb(&state, casterAetherCatalogForTest{}, "player-one", "p1-orb-1", revision); err != nil {
		t.Fatalf("ManualBanishOrb() error = %v", err)
	}
	if len(state.Players[0].Orbs) != 0 {
		t.Fatalf("Orbs = %#v; want empty", state.Players[0].Orbs)
	}
	if !slicesContains(state.Players[0].Graveyard, "p1-orb-1") {
		t.Fatalf("Graveyard = %#v; want p1-orb-1", state.Players[0].Graveyard)
	}
}

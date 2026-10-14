package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestActivateHildeWillowRestsAndOpensPendingChoice(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	state, catalog := hildeWillowStateForTest()
	enemyID, err := ActivateHildeWillow(&state, catalog, "player-one", "hilde", state.Revision)
	if err != nil {
		t.Fatalf("ActivateHildeWillow() error = %v", err)
	}
	if enemyID != "player-two" {
		t.Fatalf("enemyID = %q; want player-two", enemyID)
	}
	if state.CardInstances["hilde"].Orientation != model.OrientationRested {
		t.Fatal("Hilde Willow did not become Rested")
	}
	if state.PendingChoice.Kind != effects.PendingHildeDeckBottom {
		t.Fatalf("PendingChoice.Kind = %q; want %q", state.PendingChoice.Kind, effects.PendingHildeDeckBottom)
	}
	if state.PendingChoice.Payload != "player-two" {
		t.Fatalf("PendingChoice.Payload = %q; want player-two", state.PendingChoice.Payload)
	}
	if state.PriorityHolder != "player-one" || !state.PrioritySequenceOpen {
		t.Fatal("activator should retain priority")
	}
}

func TestActivateHildeWillowThenBottomViaPending(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	state, catalog := hildeWillowStateForTest()
	_, err := ActivateHildeWillow(&state, catalog, "player-one", "hilde", state.Revision)
	if err != nil {
		t.Fatalf("ActivateHildeWillow() error = %v", err)
	}
	if err := ResolvePendingChoice(&state, catalog, "player-one", "bottom", nil, state.Revision); err != nil {
		t.Fatalf("ResolvePendingChoice(bottom) error = %v", err)
	}
	want := []model.MatchCardID{"e2", "e3", "e1"}
	if got := state.Players[1].Deck; len(got) != 3 || got[0] != "e2" || got[2] != "e1" {
		t.Fatalf("enemy Deck = %#v; want %#v", got, want)
	}
}

func TestActivateHildeWillowRejectsWithoutPriority(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	state, catalog := hildeWillowStateForTest()
	state.PriorityHolder = "player-two"
	_, err := ActivateHildeWillow(&state, catalog, "player-one", "hilde", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "priority holder") {
		t.Fatalf("ActivateHildeWillow() error = %v; want priority holder", err)
	}
}

func TestActivateHildeWillowRejectsEmptyEnemyDeck(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	state, catalog := hildeWillowStateForTest()
	state.Players[1].Deck = nil
	_, err := ActivateHildeWillow(&state, catalog, "player-one", "hilde", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "enemy deck is empty") {
		t.Fatalf("ActivateHildeWillow() error = %v; want empty deck", err)
	}
}

func hildeWillowStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"hilde": {
				CardID:       "146",
				MatchID:      "hilde",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
				Face:         model.CardFaceUp,
				Orientation:  model.OrientationRecovered,
			},
			"e1": {CardID: "c1", MatchID: "e1", Owner: "player-two", Controller: "player-two", CardCategory: model.CategoryPrintedCard},
			"e2": {CardID: "c2", MatchID: "e2", Owner: "player-two", Controller: "player-two", CardCategory: model.CategoryPrintedCard},
			"e3": {CardID: "c3", MatchID: "e3", Owner: "player-two", Controller: "player-two", CardCategory: model.CategoryPrintedCard},
		},
		Players: [2]model.PlayerState{
			{
				ID:         "player-one",
				CasterZone: []model.MatchCardID{"hilde"},
			},
			{
				ID:   "player-two",
				Deck: []model.MatchCardID{"e1", "e2", "e3"},
			},
		},
		FirstPlayer:          "player-one",
		MatchStatus:          model.StatusInProgress,
		Revision:             1,
		Turn:                 model.TurnState{Number: 2, ActivePlayer: "player-two", Phase: model.PhaseMain},
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		NextLinkID:           1,
	}
	catalog := casterAetherCatalogForTest{
		"146": {
			ID:      "146",
			Name:    "Hilde Willow",
			Type:    "Caster",
			Ability: "• Rest: Look at the top card of an enemy deck. You may put that card on the bottom of that enemy deck.",
		},
		"c1": {ID: "c1", Name: "One", Type: "Servant"},
		"c2": {ID: "c2", Name: "Two", Type: "Servant"},
		"c3": {ID: "c3", Name: "Three", Type: "Servant"},
	}
	return state, catalog
}

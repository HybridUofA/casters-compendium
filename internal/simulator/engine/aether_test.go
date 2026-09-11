package engine

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestGenerateNonElementalAetherRestsCasterAndUpdatesPool(t *testing.T) {
	state := aetherEngineStateForTest()
	beforePlayerOne := state.Players[0]
	selectedID := model.MatchCardID("p2-caster")

	if err := GenerateNonElementalAether(&state, "player-two", selectedID, state.Revision); err != nil {
		t.Fatalf("GenerateNonElementalAether() error = %v; want nil", err)
	}
	if state.CardInstances[selectedID].Orientation != model.OrientationRested {
		t.Fatalf("Caster orientation = %q; want Rested", state.CardInstances[selectedID].Orientation)
	}
	if state.Players[1].Aether.NonElemental != 3 {
		t.Fatalf("non-elemental Aether = %d; want 3", state.Players[1].Aether.NonElemental)
	}
	if state.Revision != 6 {
		t.Fatalf("revision = %d; want 6", state.Revision)
	}
	if state.Turn.ActivePlayer != "player-one" || state.Turn.Phase != model.PhaseBattle {
		t.Fatalf("non-active player's Aether action changed turn state: %#v", state.Turn)
	}
	if !reflect.DeepEqual(state.Players[0], beforePlayerOne) {
		t.Fatal("non-active player's Aether action changed the active player's state")
	}
}

func TestGenerateNonElementalAetherRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := aetherEngineStateForTest()
	before := cloneAetherEngineState(state)

	err := GenerateNonElementalAether(&state, "player-two", "p2-caster", state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("GenerateNonElementalAether() error = %v; want stale-revision error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale Aether action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestGenerateNonElementalAetherPropagatesRuleFailureWithoutMutation(t *testing.T) {
	state := aetherEngineStateForTest()
	instance := state.CardInstances["p2-caster"]
	instance.Orientation = model.OrientationRested
	state.CardInstances["p2-caster"] = instance
	before := cloneAetherEngineState(state)

	err := GenerateNonElementalAether(&state, "player-two", "p2-caster", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "already rested") {
		t.Fatalf("GenerateNonElementalAether() error = %v; want rested-Caster error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected Aether action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestGenerateNonElementalAetherRejectsNilState(t *testing.T) {
	err := GenerateNonElementalAether(nil, "player-one", "p1-caster", 0)
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("GenerateNonElementalAether(nil) error = %v; want nil-state error", err)
	}
}

func aetherEngineStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-caster": aetherEngineCard("p1-caster", "player-one"),
			"p2-caster": aetherEngineCard("p2-caster", "player-two"),
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"p1-caster"}},
			{
				ID:         "player-two",
				CasterZone: []model.MatchCardID{"p2-caster"},
				Aether:     model.AetherPool{NonElemental: 2},
			},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    5,
		Turn: model.TurnState{
			Number:       3,
			ActivePlayer: "player-one",
			Phase:        model.PhaseBattle,
		},
	}
}

func aetherEngineCard(matchID model.MatchCardID, playerID model.PlayerID) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CardID("definition-" + string(matchID)),
		MatchID:      matchID,
		Owner:        playerID,
		Controller:   playerID,
		CardCategory: model.CategoryPrintedCard,
		Face:         model.CardFaceDown,
		Orientation:  model.OrientationRecovered,
	}
}

func cloneAetherEngineState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].Deck = slices.Clone(state.Players[index].Deck)
		clone.Players[index].Hand = slices.Clone(state.Players[index].Hand)
		clone.Players[index].Orbs = slices.Clone(state.Players[index].Orbs)
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].ServantZone = slices.Clone(state.Players[index].ServantZone)
		clone.Players[index].Graveyard = slices.Clone(state.Players[index].Graveyard)
	}
	return clone
}

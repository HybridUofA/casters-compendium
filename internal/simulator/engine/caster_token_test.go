package engine

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestUseCasterTokenRemovesTokenAndProducesNonElementalAether(t *testing.T) {
	state := casterTokenEngineStateForTest()
	beforePlayerOne := state.Players[0]

	if err := UseCasterToken(&state, "player-two", "p2-token", state.Revision); err != nil {
		t.Fatalf("UseCasterToken() error = %v; want nil", err)
	}
	if len(state.Players[1].CasterZone) != 0 {
		t.Fatalf("Caster Zone = %v; want token removed", state.Players[1].CasterZone)
	}
	if _, exists := state.CardInstances["p2-token"]; exists {
		t.Fatal("used Caster Token remained in CardInstances")
	}
	if len(state.Players[1].Exile) != 0 {
		t.Fatalf("Exile = %v; used token must cease to exist", state.Players[1].Exile)
	}
	if state.Players[1].Aether.NonElemental != 3 || state.Revision != 6 {
		t.Fatalf("Aether/revision = %d/%d; want 3/6", state.Players[1].Aether.NonElemental, state.Revision)
	}
	if state.Turn.ActivePlayer != "player-one" || state.Turn.Phase != model.PhaseBattle {
		t.Fatalf("non-active player's token action changed turn state: %#v", state.Turn)
	}
	if !reflect.DeepEqual(state.Players[0], beforePlayerOne) {
		t.Fatal("non-active player's token action changed the active player's state")
	}
}

func TestUseCasterTokenRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := casterTokenEngineStateForTest()
	before := cloneCasterTokenEngineState(state)

	err := UseCasterToken(&state, "player-two", "p2-token", state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("UseCasterToken() error = %v; want stale-revision error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale token action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestUseCasterTokenPropagatesRuleFailureWithoutMutation(t *testing.T) {
	state := casterTokenEngineStateForTest()
	instance := state.CardInstances["p2-token"]
	instance.Face = model.CardFaceDown
	state.CardInstances["p2-token"] = instance
	before := cloneCasterTokenEngineState(state)

	err := UseCasterToken(&state, "player-two", "p2-token", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "must be face-up") {
		t.Fatalf("UseCasterToken() error = %v; want face-up error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected token action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestUseCasterTokenRejectsNilState(t *testing.T) {
	err := UseCasterToken(nil, "player-one", "p1-token", 0)
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("UseCasterToken(nil) error = %v; want nil-state error", err)
	}
}

func casterTokenEngineStateForTest() model.MatchState {
	state := validCasterTokenRuleStateForEngineTest()
	state.Revision = 5
	state.Players[1].Aether.NonElemental = 2
	return state
}

func validCasterTokenRuleStateForEngineTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-token": casterTokenEngineCard("p1-token", "player-one"),
			"p2-token": casterTokenEngineCard("p2-token", "player-two"),
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"p1-token"}},
			{ID: "player-two", CasterZone: []model.MatchCardID{"p2-token"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn:        model.TurnState{Number: 3, ActivePlayer: "player-one", Phase: model.PhaseBattle},
	}
}

func casterTokenEngineCard(matchID model.MatchCardID, playerID model.PlayerID) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CasterTokenCardID,
		MatchID:      matchID,
		Owner:        playerID,
		Controller:   playerID,
		CardCategory: model.CategoryTokenCard,
		Face:         model.CardFaceUp,
		Orientation:  model.OrientationRecovered,
	}
}

func cloneCasterTokenEngineState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].Exile = slices.Clone(state.Players[index].Exile)
	}
	return clone
}

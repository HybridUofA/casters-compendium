package engine

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type casterAetherCatalogForTest map[string]gamecards.Card

func (catalog casterAetherCatalogForTest) FindByID(id string) (gamecards.Card, bool) {
	card, found := catalog[id]
	return card, found
}

func TestGenerateCasterAetherUpdatesEveryElementalPool(t *testing.T) {
	tests := []struct {
		element string
		want    model.AetherPool
	}{
		{element: "Aes", want: model.AetherPool{Aes: 3}},
		{element: "Aqua", want: model.AetherPool{Aqua: 3}},
		{element: "Ignus", want: model.AetherPool{Ignus: 3}},
		{element: "Luna", want: model.AetherPool{Luna: 3}},
		{element: "Silva", want: model.AetherPool{Silva: 3}},
		{element: "Solis", want: model.AetherPool{Solis: 3}},
		{element: "Terra", want: model.AetherPool{Terra: 3}},
		{element: "Void", want: model.AetherPool{Void: 3}},
	}

	for _, testCase := range tests {
		t.Run(testCase.element, func(t *testing.T) {
			state, catalog := casterAetherEngineStateForTest(testCase.element)
			beforePlayerOne := state.Players[0]

			err := GenerateCasterAether(
				&state,
				catalog,
				"player-two",
				"p2-caster",
				state.Revision,
			)
			if err != nil {
				t.Fatalf("GenerateCasterAether() error = %v; want nil", err)
			}
			if state.Players[1].Aether != testCase.want {
				t.Fatalf("Aether pool = %#v; want %#v", state.Players[1].Aether, testCase.want)
			}
			if state.CardInstances["p2-caster"].Orientation != model.OrientationRested {
				t.Fatal("generated Caster did not become Rested")
			}
			if state.Revision != 10 {
				t.Fatalf("revision = %d; want 10", state.Revision)
			}
			if state.Turn.ActivePlayer != "player-one" || state.Turn.Phase != model.PhaseBattle {
				t.Fatalf("non-active player's action changed turn state: %#v", state.Turn)
			}
			if !reflect.DeepEqual(state.Players[0], beforePlayerOne) {
				t.Fatal("non-active player's action changed the active player")
			}
		})
	}
}

func TestGenerateCasterAetherRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state, catalog := casterAetherEngineStateForTest("Aqua")
	before := cloneCasterAetherEngineState(state)

	err := GenerateCasterAether(&state, catalog, "player-two", "p2-caster", state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("GenerateCasterAether() error = %v; want stale-revision error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale Caster action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestGenerateCasterAetherPropagatesValidationFailureWithoutMutation(t *testing.T) {
	state, catalog := casterAetherEngineStateForTest("Aqua")
	instance := state.CardInstances["p2-caster"]
	instance.Orientation = model.OrientationRested
	state.CardInstances["p2-caster"] = instance
	before := cloneCasterAetherEngineState(state)

	err := GenerateCasterAether(&state, catalog, "player-two", "p2-caster", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "must be recovered") {
		t.Fatalf("GenerateCasterAether() error = %v; want Recovered error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected Caster action mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestGenerateCasterAetherRejectsNilState(t *testing.T) {
	err := GenerateCasterAether(nil, casterAetherCatalogForTest{}, "player-one", "caster", 0)
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("GenerateCasterAether(nil) error = %v; want nil-state error", err)
	}
}

func casterAetherEngineStateForTest(element string) (model.MatchState, casterAetherCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p2-caster": {
				CardID:       "level-three-caster",
				MatchID:      "p2-caster",
				Owner:        "player-one",
				Controller:   "player-two",
				CardCategory: model.CategoryPrintedCard,
				Face:         model.CardFaceUp,
				Orientation:  model.OrientationRecovered,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two", CasterZone: []model.MatchCardID{"p2-caster"}},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    9,
		Turn: model.TurnState{
			Number:       4,
			ActivePlayer: "player-one",
			Phase:        model.PhaseBattle,
		},
	}
	catalog := casterAetherCatalogForTest{
		"level-three-caster": {
			ID:        "level-three-caster",
			Type:      "Caster",
			Element:   element,
			CostLevel: "3",
		},
	}
	return state, catalog
}

func cloneCasterAetherEngineState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].Exile = slices.Clone(state.Players[index].Exile)
	}
	return clone
}

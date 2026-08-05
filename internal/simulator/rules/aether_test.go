package rules

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateGenerateNonElementalAetherAllowsEitherPlayerWithoutMutation(t *testing.T) {
	state := validAetherRuleState()
	before := cloneAetherRuleState(state)

	if err := ValidateGenerateNonElementalAether(&state, "player-two", "p2-caster"); err != nil {
		t.Fatalf("ValidateGenerateNonElementalAether() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateGenerateNonElementalAetherRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name         string
		actingPlayer model.PlayerID
		cardID       model.MatchCardID
		mutate       func(*model.MatchState)
		wantErrPart  string
	}{
		{
			name:         "match is not in progress",
			actingPlayer: "player-two",
			cardID:       "p2-caster",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusSetup
			},
			wantErrPart: "expected \"In Progress\"",
		},
		{
			name:         "blank acting player",
			actingPlayer: " ",
			cardID:       "p2-caster",
			wantErrPart:  "player ID cannot be empty",
		},
		{
			name:         "unknown acting player",
			actingPlayer: "missing-player",
			cardID:       "p2-caster",
			wantErrPart:  "acting player ID not found",
		},
		{
			name:         "card is not in player's Caster Zone",
			actingPlayer: "player-two",
			cardID:       "unplaced-card",
			mutate: func(state *model.MatchState) {
				state.CardInstances["unplaced-card"] = aetherRuleCard("unplaced-card", "player-two")
			},
			wantErrPart: "must exist in caster zone",
		},
		{
			name:         "Caster Zone card is absent from instances",
			actingPlayer: "player-two",
			cardID:       "p2-caster",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p2-caster")
			},
			wantErrPart: "does not exist in card instances",
		},
		{
			name:         "card is face up",
			actingPlayer: "player-two",
			cardID:       "p2-caster",
			mutate: func(state *model.MatchState) {
				instance := state.CardInstances["p2-caster"]
				instance.Face = model.CardFaceUp
				state.CardInstances["p2-caster"] = instance
			},
			wantErrPart: "card is face up",
		},
		{
			name:         "card is not Recovered",
			actingPlayer: "player-two",
			cardID:       "p2-caster",
			mutate: func(state *model.MatchState) {
				instance := state.CardInstances["p2-caster"]
				instance.Orientation = model.OrientationRested
				state.CardInstances["p2-caster"] = instance
			},
			wantErrPart: "already rested",
		},
		{
			name:         "card is controlled by the other player",
			actingPlayer: "player-two",
			cardID:       "p2-caster",
			mutate: func(state *model.MatchState) {
				instance := state.CardInstances["p2-caster"]
				instance.Controller = "player-one"
				state.CardInstances["p2-caster"] = instance
			},
			wantErrPart: "do not control",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := validAetherRuleState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneAetherRuleState(state)

			err := ValidateGenerateNonElementalAether(&state, testCase.actingPlayer, testCase.cardID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateGenerateNonElementalAether() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateGenerateNonElementalAetherRejectsNilState(t *testing.T) {
	err := ValidateGenerateNonElementalAether(nil, "player-one", "p1-caster")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateGenerateNonElementalAether(nil) error = %v; want nil-state error", err)
	}
}

func validAetherRuleState() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-caster": aetherRuleCard("p1-caster", "player-one"),
			"p2-caster": aetherRuleCard("p2-caster", "player-two"),
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"p1-caster"}},
			{ID: "player-two", CasterZone: []model.MatchCardID{"p2-caster"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       3,
			ActivePlayer: "player-one",
			Phase:        model.PhaseBattle,
		},
	}
}

func aetherRuleCard(matchID model.MatchCardID, playerID model.PlayerID) model.CardInstance {
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

func cloneAetherRuleState(state model.MatchState) model.MatchState {
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

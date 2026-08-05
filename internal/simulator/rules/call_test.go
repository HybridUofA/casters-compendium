package rules

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateFaceDownLevelOneCallAcceptsLegalActionWithoutMutation(t *testing.T) {
	state := validFaceDownLevelOneCallState()
	before := cloneCallTestState(state)

	if err := ValidateFaceDownLevelOneCall(&state, "player-one", "p1-hand-card"); err != nil {
		t.Fatalf("ValidateFaceDownLevelOneCall() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateFaceDownLevelOneCallRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name         string
		actingPlayer model.PlayerID
		cardID       model.MatchCardID
		mutate       func(*model.MatchState)
		wantErrPart  string
	}{
		{
			name:         "match is not in progress",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusSetup
			},
			wantErrPart: "expected \"In Progress\"",
		},
		{
			name:         "not in Call phase",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.Phase = model.PhaseMain
			},
			wantErrPart: "expected \"Call\"",
		},
		{
			name:         "blank acting player",
			actingPlayer: "  ",
			cardID:       "p1-hand-card",
			wantErrPart:  "player ID cannot be empty",
		},
		{
			name:         "acting player is not active",
			actingPlayer: "player-two",
			cardID:       "p2-hand-card",
			wantErrPart:  "is not the active player",
		},
		{
			name:         "Call action was already taken",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.CallActionTaken = true
			},
			wantErrPart: "call action has already been taken",
		},
		{
			name:         "active player is absent from players",
			actingPlayer: "missing-player",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.ActivePlayer = "missing-player"
			},
			wantErrPart: "acting player ID not found",
		},
		{
			name:         "card is not in active player's hand",
			actingPlayer: "player-one",
			cardID:       "p2-hand-card",
			wantErrPart:  "card not found in hand",
		},
		{
			name:         "hand card is absent from card instances",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p1-hand-card")
			},
			wantErrPart: "not found in card instances",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := validFaceDownLevelOneCallState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneCallTestState(state)

			err := ValidateFaceDownLevelOneCall(
				&state,
				testCase.actingPlayer,
				testCase.cardID,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateFaceDownLevelOneCall() error = %v; want error containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateFaceDownLevelOneCallRejectsNilState(t *testing.T) {
	err := ValidateFaceDownLevelOneCall(nil, "player-one", "p1-hand-card")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateFaceDownLevelOneCall(nil) error = %v; want nil-state error", err)
	}
}

func validFaceDownLevelOneCallState() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-hand-card": {
				CardID:       "definition-one",
				MatchID:      "p1-hand-card",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
			"p2-hand-card": {
				CardID:       "definition-two",
				MatchID:      "p2-hand-card",
				Owner:        "player-two",
				Controller:   "player-two",
				CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Hand: []model.MatchCardID{"p1-hand-card"}},
			{ID: "player-two", Hand: []model.MatchCardID{"p2-hand-card"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseCall,
		},
	}
}

func cloneCallTestState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].Deck = slices.Clone(state.Players[index].Deck)
		clone.Players[index].Hand = slices.Clone(state.Players[index].Hand)
		clone.Players[index].Orbs = slices.Clone(state.Players[index].Orbs)
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
	}
	return clone
}

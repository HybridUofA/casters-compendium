package engine

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestCallFaceDownLevelOneMovesSelectedCardAndUpdatesTurnState(t *testing.T) {
	state := callStateForTest()
	selectedID := model.MatchCardID("p1-hand-b")
	beforeInstance := state.CardInstances[selectedID]

	if err := CallFaceDownLevelOne(&state, "player-one", selectedID, state.Revision); err != nil {
		t.Fatalf("CallFaceDownLevelOne() error = %v; want nil", err)
	}

	if !reflect.DeepEqual(state.Players[0].Hand, []model.MatchCardID{"p1-hand-a"}) {
		t.Fatalf("active player's hand = %v; want selected card removed", state.Players[0].Hand)
	}
	if !reflect.DeepEqual(state.Players[0].CasterZone, []model.MatchCardID{"p1-token", selectedID}) {
		t.Fatalf("active player's Caster Zone = %v; want token followed by %q", state.Players[0].CasterZone, selectedID)
	}
	calledInstance := state.CardInstances[selectedID]
	if calledInstance.Face != model.CardFaceDown ||
		calledInstance.Orientation != model.OrientationRecovered ||
		calledInstance.Controller != "player-one" {
		t.Fatalf("called card state = face %q, orientation %q, controller %q; want face down, recovered, player-one", calledInstance.Face, calledInstance.Orientation, calledInstance.Controller)
	}
	if calledInstance.MatchID != beforeInstance.MatchID ||
		calledInstance.CardID != beforeInstance.CardID ||
		calledInstance.Owner != beforeInstance.Owner ||
		calledInstance.CardCategory != beforeInstance.CardCategory {
		t.Fatalf("Call changed stable card identity fields\n before: %#v\n  after: %#v", beforeInstance, calledInstance)
	}
	if !state.Turn.CallActionTaken {
		t.Fatal("Call action was not recorded in turn state")
	}
	if state.Revision != 8 {
		t.Fatalf("revision = %d; want 8", state.Revision)
	}
	if state.Turn.Phase != model.PhaseCall || state.Turn.ActivePlayer != "player-one" {
		t.Fatalf("Call changed phase or active player: %#v", state.Turn)
	}
	if !reflect.DeepEqual(state.Players[1], callStateForTest().Players[1]) {
		t.Fatal("Call changed the opponent's state")
	}
}

func TestCallFaceDownLevelOneRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := callStateForTest()
	before := cloneCallStateForTest(state)

	err := CallFaceDownLevelOne(&state, "player-one", "p1-hand-a", state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("CallFaceDownLevelOne() error = %v; want stale-revision error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale Call mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestCallFaceDownLevelOnePropagatesRuleRejectionsWithoutMutation(t *testing.T) {
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
			cardID:       "p1-hand-a",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusSetup
			},
			wantErrPart: "expected \"In Progress\"",
		},
		{
			name:         "not in Call phase",
			actingPlayer: "player-one",
			cardID:       "p1-hand-a",
			mutate: func(state *model.MatchState) {
				state.Turn.Phase = model.PhaseMain
			},
			wantErrPart: "expected \"Call\"",
		},
		{
			name:         "blank acting player",
			actingPlayer: " ",
			cardID:       "p1-hand-a",
			wantErrPart:  "player ID cannot be empty",
		},
		{
			name:         "non-active player",
			actingPlayer: "player-two",
			cardID:       "p2-hand-a",
			wantErrPart:  "is not the active player",
		},
		{
			name:         "Call action already taken",
			actingPlayer: "player-one",
			cardID:       "p1-hand-a",
			mutate: func(state *model.MatchState) {
				state.Turn.CallActionTaken = true
			},
			wantErrPart: "call action has already been taken",
		},
		{
			name:         "active player absent from player state",
			actingPlayer: "missing-player",
			cardID:       "p1-hand-a",
			mutate: func(state *model.MatchState) {
				state.Turn.ActivePlayer = "missing-player"
			},
			wantErrPart: "acting player ID not found",
		},
		{
			name:         "card not in active player's hand",
			actingPlayer: "player-one",
			cardID:       "p2-hand-a",
			wantErrPart:  "card not found in hand",
		},
		{
			name:         "hand card missing from instances",
			actingPlayer: "player-one",
			cardID:       "p1-hand-a",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p1-hand-a")
			},
			wantErrPart: "not found in card instances",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := callStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneCallStateForTest(state)

			err := CallFaceDownLevelOne(
				&state,
				testCase.actingPlayer,
				testCase.cardID,
				state.Revision,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("CallFaceDownLevelOne() error = %v; want error containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected Call mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestCallFaceDownLevelOneRejectsNilState(t *testing.T) {
	err := CallFaceDownLevelOne(nil, "player-one", "p1-hand-a", 0)
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("CallFaceDownLevelOne(nil) error = %v; want nil-state error", err)
	}
}

func callStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-token": {
				CardID:       model.CasterTokenCardID,
				MatchID:      "p1-token",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryTokenCard,
				Face:         model.CardFaceUp,
				Orientation:  model.OrientationRecovered,
			},
			"p1-hand-a": callTestPrintedCard("p1-hand-a", "player-one", model.OrientationRested),
			"p1-hand-b": callTestPrintedCard("p1-hand-b", "player-one", model.OrientationReversed),
			"p2-hand-a": callTestPrintedCard("p2-hand-a", "player-two", model.OrientationRecovered),
		},
		Players: [2]model.PlayerState{
			{
				ID:         "player-one",
				Hand:       []model.MatchCardID{"p1-hand-a", "p1-hand-b"},
				CasterZone: []model.MatchCardID{"p1-token"},
			},
			{
				ID:   "player-two",
				Hand: []model.MatchCardID{"p2-hand-a"},
			},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusInProgress,
		Revision:    7,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseCall,
		},
	}
}

func callTestPrintedCard(
	matchID model.MatchCardID,
	owner model.PlayerID,
	orientation model.CardOrientation,
) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CardID("definition-" + string(matchID)),
		MatchID:      matchID,
		Owner:        owner,
		Controller:   owner,
		CardCategory: model.CategoryPrintedCard,
		Face:         model.CardFaceUp,
		Orientation:  orientation,
	}
}

func cloneCallStateForTest(state model.MatchState) model.MatchState {
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

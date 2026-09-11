package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestCompleteRecoveryPhaseSkipsFirstPlayersFirstDraw(t *testing.T) {
	state := initialRecoveryStateForTest()
	beforeHands := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Hand...),
		append([]model.MatchCardID(nil), state.Players[1].Hand...),
	}
	beforeDecks := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Deck...),
		append([]model.MatchCardID(nil), state.Players[1].Deck...),
	}

	if err := completeRecoveryPhase(&state, state.Turn.ActivePlayer); err != nil {
		t.Fatalf("completeRecoveryPhase() error = %v", err)
	}

	if state.Turn.Phase != model.PhaseCall {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseCall)
	}
	if !reflect.DeepEqual(state.Players[0].Hand, beforeHands[0]) ||
		!reflect.DeepEqual(state.Players[1].Hand, beforeHands[1]) ||
		!reflect.DeepEqual(state.Players[0].Deck, beforeDecks[0]) ||
		!reflect.DeepEqual(state.Players[1].Deck, beforeDecks[1]) {
		t.Fatal("initial Recovery completion drew or moved cards")
	}
}

func TestCompleteRecoveryPhaseAdvancesLaterTurnsToDraw(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Number = 2
	state.Turn.ActivePlayer = state.Players[1].ID
	topCard := state.Players[1].Deck[0]
	beforeDeckSize := len(state.Players[1].Deck)
	beforeHandSize := len(state.Players[1].Hand)

	if err := completeRecoveryPhase(&state, state.Turn.ActivePlayer); err != nil {
		t.Fatalf("completeRecoveryPhase() error = %v", err)
	}
	if state.Turn.Phase != model.PhaseDraw {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseDraw)
	}
	if len(state.Players[1].Deck) != beforeDeckSize-1 {
		t.Fatalf("deck size = %d; want %d", len(state.Players[1].Deck), beforeDeckSize-1)
	}
	if len(state.Players[1].Hand) != beforeHandSize+1 {
		t.Fatalf("hand size = %d; want %d", len(state.Players[1].Hand), beforeHandSize+1)
	}
	if drawn := state.Players[1].Hand[len(state.Players[1].Hand)-1]; drawn != topCard {
		t.Fatalf("drawn card = %q; want former deck top %q", drawn, topCard)
	}
}

func TestCompleteCurrentPhaseCompletesDrawWithoutDrawingAgain(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Number = 2
	state.Turn.ActivePlayer = state.Players[1].ID

	if err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase(Recovery) error = %v", err)
	}
	beforeHands := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Hand...),
		append([]model.MatchCardID(nil), state.Players[1].Hand...),
	}
	beforeDecks := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Deck...),
		append([]model.MatchCardID(nil), state.Players[1].Deck...),
	}

	if err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase(Draw) error = %v", err)
	}
	if state.Turn.Phase != model.PhaseCall {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseCall)
	}
	if !reflect.DeepEqual(state.Players[0].Hand, beforeHands[0]) ||
		!reflect.DeepEqual(state.Players[1].Hand, beforeHands[1]) ||
		!reflect.DeepEqual(state.Players[0].Deck, beforeDecks[0]) ||
		!reflect.DeepEqual(state.Players[1].Deck, beforeDecks[1]) {
		t.Fatal("completing Draw moved another card")
	}
}

func TestCompleteCurrentPhaseRejectsEmptyDeckWithoutMutation(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Number = 2
	state.Turn.ActivePlayer = state.Players[1].ID
	state.Players[1].Deck = nil
	before := state

	err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "deck cannot be empty") {
		t.Fatalf("CompleteCurrentPhase() error = %v; want empty-deck error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("failed Draw entry mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestCompleteCurrentPhaseRejectsDrawOnTurnOneWithoutMutation(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Phase = model.PhaseDraw
	before := state

	err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "turn 1") {
		t.Fatalf("CompleteCurrentPhase() error = %v; want turn-one Draw error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected turn-one Draw mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestCompleteCurrentPhaseCompletesCallWithoutMovingCards(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Phase = model.PhaseCall
	beforeHands := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Hand...),
		append([]model.MatchCardID(nil), state.Players[1].Hand...),
	}
	beforeDecks := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Deck...),
		append([]model.MatchCardID(nil), state.Players[1].Deck...),
	}

	if err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase(Call) error = %v", err)
	}
	if state.Turn.Phase != model.PhaseMain {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseMain)
	}
	if !reflect.DeepEqual(state.Players[0].Hand, beforeHands[0]) ||
		!reflect.DeepEqual(state.Players[1].Hand, beforeHands[1]) ||
		!reflect.DeepEqual(state.Players[0].Deck, beforeDecks[0]) ||
		!reflect.DeepEqual(state.Players[1].Deck, beforeDecks[1]) {
		t.Fatal("completing Call moved cards")
	}
}

func TestCompleteCallPhaseRejectsWrongCurrentPhaseWithoutMutation(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Phase = model.PhaseDraw
	state.Turn.Number = 2
	before := state

	err := completeCallPhase(&state, state.Turn.ActivePlayer)
	if err == nil || !strings.Contains(err.Error(), `"Call" phase`) {
		t.Fatalf("completeCallPhase() error = %v; want Call-phase error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected Call completion mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestCompleteCurrentPhaseRunsRemainingSkeletonAndRollsTurn(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Phase = model.PhaseCall
	state.Players[0].Aether = model.AetherPool{Aes: 2, Void: 1}
	state.Players[1].Aether = model.AetherPool{Aqua: 1, NonElemental: 3}
	beforeAether := [2]model.AetherPool{
		state.Players[0].Aether,
		state.Players[1].Aether,
	}
	firstPlayer := state.Turn.ActivePlayer
	secondPlayer := state.Players[1].ID
	beforeHands := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Hand...),
		append([]model.MatchCardID(nil), state.Players[1].Hand...),
	}
	beforeDecks := [2][]model.MatchCardID{
		append([]model.MatchCardID(nil), state.Players[0].Deck...),
		append([]model.MatchCardID(nil), state.Players[1].Deck...),
	}

	for step, wantPhase := range []model.Phase{
		model.PhaseMain,
		model.PhaseBattle,
		model.PhaseEnd,
	} {
		if err := CompleteCurrentPhase(&state, firstPlayer, state.Revision); err != nil {
			t.Fatalf("CompleteCurrentPhase() toward %q error = %v", wantPhase, err)
		}
		if state.Turn.Phase != wantPhase {
			t.Fatalf("phase = %q; want %q", state.Turn.Phase, wantPhase)
		}
		if state.Turn.ActivePlayer != firstPlayer || state.Turn.Number != 1 {
			t.Fatalf("phase %q prematurely rolled turn: active=%q turn=%d", wantPhase, state.Turn.ActivePlayer, state.Turn.Number)
		}
		wantRevision := model.Revision(step + 1)
		if state.Revision != wantRevision {
			t.Fatalf("revision at phase %q = %d; want %d", wantPhase, state.Revision, wantRevision)
		}
		if state.Players[0].Aether != beforeAether[0] ||
			state.Players[1].Aether != beforeAether[1] {
			t.Fatalf("Aether cleared before End completion at phase %q", wantPhase)
		}
	}

	state.Turn.CallActionTaken = true
	if err := CompleteCurrentPhase(&state, firstPlayer, state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase(End) error = %v", err)
	}
	if state.Turn.Phase != model.PhaseRecovery ||
		state.Turn.ActivePlayer != secondPlayer ||
		state.Turn.Number != 2 ||
		state.Turn.CallActionTaken ||
		state.Players[0].Aether != (model.AetherPool{}) ||
		state.Players[1].Aether != (model.AetherPool{}) ||
		state.Revision != 4 {
		t.Fatalf("rollover state = phase %q, active %q, turn %d, Call action taken %t, Aether %#v/%#v, revision %d; want Recovery, %q, 2, false, empty pools, revision 4", state.Turn.Phase, state.Turn.ActivePlayer, state.Turn.Number, state.Turn.CallActionTaken, state.Players[0].Aether, state.Players[1].Aether, state.Revision, secondPlayer)
	}
	if !reflect.DeepEqual(state.Players[0].Hand, beforeHands[0]) ||
		!reflect.DeepEqual(state.Players[1].Hand, beforeHands[1]) ||
		!reflect.DeepEqual(state.Players[0].Deck, beforeDecks[0]) ||
		!reflect.DeepEqual(state.Players[1].Deck, beforeDecks[1]) {
		t.Fatal("phase skeleton or rollover moved cards")
	}
}

func TestCompleteEndPhaseRejectsMissingActivePlayerWithoutClearingAether(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Turn.Phase = model.PhaseEnd
	state.Turn.ActivePlayer = "missing-player"
	state.Players[0].Aether = model.AetherPool{Ignus: 2}
	state.Players[1].Aether = model.AetherPool{Terra: 1, NonElemental: 2}
	before := state

	err := completeEndPhase(&state, "missing-player")
	if err == nil || !strings.Contains(err.Error(), "not found in players") {
		t.Fatalf("completeEndPhase() error = %v; want missing-player error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("rejected End completion mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestCompleteRecoveryPhaseRejectsInvalidStateWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*model.MatchState) model.PlayerID
		wantErrPart string
	}{
		{
			name: "empty acting player",
			mutate: func(*model.MatchState) model.PlayerID {
				return ""
			},
			wantErrPart: "player ID cannot be empty",
		},
		{
			name: "wrong match status",
			mutate: func(state *model.MatchState) model.PlayerID {
				state.MatchStatus = model.StatusSetup
				return state.Turn.ActivePlayer
			},
			wantErrPart: "must be",
		},
		{
			name: "zero turn count",
			mutate: func(state *model.MatchState) model.PlayerID {
				state.Turn.Number = 0
				return state.Turn.ActivePlayer
			},
			wantErrPart: "cannot be less than 1",
		},
		{
			name: "negative turn count",
			mutate: func(state *model.MatchState) model.PlayerID {
				state.Turn.Number = -1
				return state.Turn.ActivePlayer
			},
			wantErrPart: "cannot be less than 1",
		},
		{
			name: "wrong phase",
			mutate: func(state *model.MatchState) model.PlayerID {
				state.Turn.Phase = model.PhaseDraw
				return state.Turn.ActivePlayer
			},
			wantErrPart: `"Recovery" phase`,
		},
		{
			name: "non-active player",
			mutate: func(state *model.MatchState) model.PlayerID {
				return state.Players[1].ID
			},
			wantErrPart: "not the current active player",
		},
		{
			name: "active player differs from first player",
			mutate: func(state *model.MatchState) model.PlayerID {
				state.FirstPlayer = state.Players[1].ID
				return state.Turn.ActivePlayer
			},
			wantErrPart: "not set as the first player",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := initialRecoveryStateForTest()
			actingPlayer := test.mutate(&state)
			before := state

			err := completeRecoveryPhase(&state, actingPlayer)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("completeRecoveryPhase() error = %v; want containing %q", err, test.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected transition mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestCompleteRecoveryPhaseRejectsNilState(t *testing.T) {
	err := completeRecoveryPhase(nil, "player-one")
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("completeRecoveryPhase(nil) error = %v; want nil-state error", err)
	}
}

func TestCompleteCurrentPhaseSupportsInitialRecoveryToCall(t *testing.T) {
	state := initialRecoveryStateForTest()

	if err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase() error = %v", err)
	}
	if state.Turn.Phase != model.PhaseCall {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseCall)
	}
}

func TestCompleteCurrentPhaseRejectsInvalidRequestWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(*model.MatchState) model.PlayerID
		wantErrPart string
	}{
		{
			name: "blank acting player",
			prepare: func(*model.MatchState) model.PlayerID {
				return ""
			},
			wantErrPart: "player ID cannot be empty",
		},
		{
			name: "unsupported current phase",
			prepare: func(state *model.MatchState) model.PlayerID {
				state.Turn.Phase = model.Phase("Unknown")
				return state.Turn.ActivePlayer
			},
			wantErrPart: "unsupported or illegal transition",
		},
		{
			name: "non-active player",
			prepare: func(state *model.MatchState) model.PlayerID {
				return state.Players[1].ID
			},
			wantErrPart: "not the current active player",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := initialRecoveryStateForTest()
			actingPlayer := test.prepare(&state)
			before := state

			err := CompleteCurrentPhase(&state, actingPlayer, state.Revision)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("CompleteCurrentPhase() error = %v; want containing %q", err, test.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected phase request mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestCompleteCurrentPhaseRejectsNilState(t *testing.T) {
	err := CompleteCurrentPhase(nil, "player-one", 0)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("CompleteCurrentPhase(nil) error = %v; want nil-state error", err)
	}
}

func TestCompleteCurrentPhaseRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := initialRecoveryStateForTest()
	state.Revision = 9
	before := state

	err := CompleteCurrentPhase(&state, state.Turn.ActivePlayer, 8)
	if err == nil || !strings.Contains(err.Error(), "expected revision 8") ||
		!strings.Contains(err.Error(), "current revision 9") {
		t.Fatalf("CompleteCurrentPhase() error = %v; want stale-revision details", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale phase completion mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func initialRecoveryStateForTest() model.MatchState {
	return model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Hand: matchIDs("p1-hand", 7), Deck: matchIDs("p1-deck", 36)},
			{ID: "player-two", Hand: matchIDs("p2-hand", 7), Deck: matchIDs("p2-deck", 36)},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseRecovery,
		},
	}
}

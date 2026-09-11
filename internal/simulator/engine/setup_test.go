package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestApplyOpeningHandDecisionReplacesCardsWithoutReshuffling(t *testing.T) {
	state := openingDecisionStateForTest()
	originalHand := append([]model.MatchCardID(nil), state.Players[0].Hand...)
	originalDeck := append([]model.MatchCardID(nil), state.Players[0].Deck...)
	replaced := []model.MatchCardID{originalHand[1], originalHand[4]}

	err := ApplyOpeningHandDecision(&state, OpeningHandDecision{
		PlayerID: state.Players[0].ID,
		Replace:  replaced,
	})
	if err != nil {
		t.Fatalf("ApplyOpeningHandDecision() error = %v", err)
	}

	wantHand := []model.MatchCardID{
		originalHand[0],
		originalHand[2],
		originalHand[3],
		originalHand[5],
		originalHand[6],
		originalDeck[0],
		originalDeck[1],
	}
	if !reflect.DeepEqual(state.Players[0].Hand, wantHand) {
		t.Fatalf("replacement hand = %v; want %v", state.Players[0].Hand, wantHand)
	}
	wantDeck := append([]model.MatchCardID(nil), originalDeck[2:]...)
	wantDeck = append(wantDeck, replaced...)
	if !reflect.DeepEqual(state.Players[0].Deck, wantDeck) {
		t.Fatalf("replacement deck = %v; want %v", state.Players[0].Deck, wantDeck)
	}
	if !state.Players[0].OpeningHandFinalized {
		t.Fatal("opening hand was not finalized")
	}
	if state.Revision != 1 {
		t.Fatalf("revision = %d; want 1 after one accepted decision", state.Revision)
	}
	if state.MatchStatus != model.StatusSetup || state.Turn.Number != 0 || state.Turn.Phase != "" {
		t.Fatalf("match advanced before both players finalized: %#v", state)
	}
}

func TestApplyOpeningHandDecisionStartsFirstTurnAfterBothPlayersFinalize(t *testing.T) {
	state := openingDecisionStateForTest()

	for playerIndex := range state.Players {
		err := ApplyOpeningHandDecision(&state, OpeningHandDecision{
			PlayerID:         state.Players[playerIndex].ID,
			ExpectedRevision: state.Revision,
		})
		if err != nil {
			t.Fatalf("player %d decision error = %v", playerIndex+1, err)
		}
		wantRevision := model.Revision(playerIndex + 1)
		if state.Revision != wantRevision {
			t.Fatalf("revision after player %d = %d; want %d", playerIndex+1, state.Revision, wantRevision)
		}
	}

	if state.MatchStatus != model.StatusInProgress {
		t.Fatalf("match status = %q; want %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Number != 1 {
		t.Fatalf("turn count = %d; want 1", state.Turn.Number)
	}
	if state.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("phase = %q; want %q", state.Turn.Phase, model.PhaseRecovery)
	}
	if state.Turn.ActivePlayer != state.FirstPlayer {
		t.Fatalf("active player = %q; want first player %q", state.Turn.ActivePlayer, state.FirstPlayer)
	}
}

func TestApplyOpeningHandDecisionRejectsInvalidRequestWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(*model.MatchState) OpeningHandDecision
		wantErrPart string
	}{
		{
			name: "empty player ID",
			prepare: func(*model.MatchState) OpeningHandDecision {
				return OpeningHandDecision{}
			},
			wantErrPart: "player ID cannot be empty",
		},
		{
			name: "unknown player",
			prepare: func(*model.MatchState) OpeningHandDecision {
				return OpeningHandDecision{PlayerID: "unknown"}
			},
			wantErrPart: "unknown player ID",
		},
		{
			name: "wrong match status",
			prepare: func(state *model.MatchState) OpeningHandDecision {
				state.MatchStatus = model.StatusInProgress
				return OpeningHandDecision{PlayerID: state.Players[0].ID}
			},
			wantErrPart: "set up phase",
		},
		{
			name: "already finalized",
			prepare: func(state *model.MatchState) OpeningHandDecision {
				state.Players[0].OpeningHandFinalized = true
				return OpeningHandDecision{PlayerID: state.Players[0].ID}
			},
			wantErrPart: "already been finalized",
		},
		{
			name: "too many replacements",
			prepare: func(state *model.MatchState) OpeningHandDecision {
				return OpeningHandDecision{
					PlayerID: state.Players[0].ID,
					Replace: []model.MatchCardID{
						"a", "b", "c", "d", "e", "f", "g", "h",
					},
				}
			},
			wantErrPart: "cannot replace more cards",
		},
		{
			name: "duplicate selection",
			prepare: func(state *model.MatchState) OpeningHandDecision {
				cardID := state.Players[0].Hand[0]
				return OpeningHandDecision{
					PlayerID: state.Players[0].ID,
					Replace:  []model.MatchCardID{cardID, cardID},
				}
			},
			wantErrPart: "same card may not be selected twice",
		},
		{
			name: "card not in hand",
			prepare: func(state *model.MatchState) OpeningHandDecision {
				return OpeningHandDecision{
					PlayerID: state.Players[0].ID,
					Replace:  []model.MatchCardID{state.Players[0].Deck[0]},
				}
			},
			wantErrPart: "is not",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := openingDecisionStateForTest()
			decision := test.prepare(&state)
			before := cloneOpeningDecisionState(state)

			err := ApplyOpeningHandDecision(&state, decision)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("ApplyOpeningHandDecision() error = %v; want containing %q", err, test.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected decision mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestApplyOpeningHandDecisionRejectsNilState(t *testing.T) {
	err := ApplyOpeningHandDecision(nil, OpeningHandDecision{PlayerID: "player-one"})
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("ApplyOpeningHandDecision(nil) error = %v; want nil-state error", err)
	}
}

func TestApplyOpeningHandDecisionRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := openingDecisionStateForTest()
	state.Revision = 4
	before := cloneOpeningDecisionState(state)

	err := ApplyOpeningHandDecision(&state, OpeningHandDecision{
		PlayerID:         state.Players[0].ID,
		ExpectedRevision: 3,
	})
	if err == nil || !strings.Contains(err.Error(), "expected revision 3") ||
		!strings.Contains(err.Error(), "current revision 4") {
		t.Fatalf("ApplyOpeningHandDecision() error = %v; want stale-revision details", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("stale opening-hand decision mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func openingDecisionStateForTest() model.MatchState {
	return model.MatchState{
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: matchIDs("p1-hand", 7),
				Deck: matchIDs("p1-deck", 36),
			},
			{
				ID:   "player-two",
				Hand: matchIDs("p2-hand", 7),
				Deck: matchIDs("p2-deck", 36),
			},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusSetup,
		Turn: model.TurnState{
			ActivePlayer: "player-one",
		},
	}
}

func matchIDs(prefix string, count int) []model.MatchCardID {
	result := make([]model.MatchCardID, count)
	for index := range result {
		result[index] = model.MatchCardID(prefix + "-" + string(rune('a'+index)))
	}
	return result
}

func cloneOpeningDecisionState(state model.MatchState) model.MatchState {
	for index := range state.Players {
		state.Players[index].Deck = append([]model.MatchCardID(nil), state.Players[index].Deck...)
		state.Players[index].Hand = append([]model.MatchCardID(nil), state.Players[index].Hand...)
	}
	return state
}

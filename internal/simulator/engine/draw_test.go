package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestDrawCardsMovesTopCardsToHand(t *testing.T) {
	state := drawStateForTest()
	beforeHand := append([]model.MatchCardID(nil), state.Players[0].Hand...)
	beforeDeck := append([]model.MatchCardID(nil), state.Players[0].Deck...)

	if err := DrawCards(&state, "player-one", 2); err != nil {
		t.Fatalf("DrawCards() error = %v", err)
	}

	wantHand := append(append([]model.MatchCardID(nil), beforeHand...), beforeDeck[0], beforeDeck[1])
	if !reflect.DeepEqual(state.Players[0].Hand, wantHand) {
		t.Fatalf("Hand = %#v; want %#v", state.Players[0].Hand, wantHand)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, beforeDeck[2:]) {
		t.Fatalf("Deck = %#v; want %#v", state.Players[0].Deck, beforeDeck[2:])
	}
	if state.Revision != 3 {
		t.Fatalf("Revision = %d; DrawCards helper must not bump revision", state.Revision)
	}
	if !reflect.DeepEqual(state.Players[1].Hand, []model.MatchCardID{"p2-hand-1"}) ||
		!reflect.DeepEqual(state.Players[1].Deck, []model.MatchCardID{"p2-deck-1", "p2-deck-2"}) {
		t.Fatal("DrawCards mutated the other player")
	}
}

func TestDrawCardsRejectsInsufficientDeckWithoutMutation(t *testing.T) {
	state := drawStateForTest()
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	err = DrawCards(&state, "player-one", len(state.Players[0].Deck)+1)
	if err == nil || !strings.Contains(err.Error(), "error taking cards") {
		t.Fatalf("DrawCards() error = %v; want taking-cards error", err)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected DrawCards mutated state")
	}
}

func TestDrawCardsRejectsUnknownPlayerAndNilState(t *testing.T) {
	if err := DrawCards(nil, "player-one", 1); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("DrawCards(nil) error = %v; want nil-state error", err)
	}

	state := drawStateForTest()
	before := state
	err := DrawCards(&state, "missing", 1)
	if err == nil || !strings.Contains(err.Error(), "player not found") {
		t.Fatalf("DrawCards() error = %v; want player-not-found", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("unknown-player DrawCards mutated state")
	}
}

func TestManualDrawCardsBumpsRevision(t *testing.T) {
	state := drawStateForTest()
	beforeHandLen := len(state.Players[0].Hand)
	beforeDeckLen := len(state.Players[0].Deck)

	if err := ManualDrawCards(&state, nil, "player-one", 1, state.Revision); err != nil {
		t.Fatalf("ManualDrawCards() error = %v", err)
	}
	if len(state.Players[0].Hand) != beforeHandLen+1 {
		t.Fatalf("hand len = %d; want %d", len(state.Players[0].Hand), beforeHandLen+1)
	}
	if len(state.Players[0].Deck) != beforeDeckLen-1 {
		t.Fatalf("deck len = %d; want %d", len(state.Players[0].Deck), beforeDeckLen-1)
	}
	if state.Revision != 4 {
		t.Fatalf("Revision = %d; want 4", state.Revision)
	}
}

func TestManualDrawCardsRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*model.MatchState)
		player  model.PlayerID
		count   int
		rev     model.Revision
		wantErr string
	}{
		{
			name:    "stale revision",
			player:  "player-one",
			count:   1,
			rev:     2,
			wantErr: "expected revision 2",
		},
		{
			name:    "count zero",
			player:  "player-one",
			count:   0,
			rev:     3,
			wantErr: "at least 1",
		},
		{
			name: "match finished",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusFinished
			},
			player:  "player-one",
			count:   1,
			rev:     3,
			wantErr: "not in progress",
		},
		{
			name:    "insufficient cards",
			player:  "player-one",
			count:   99,
			rev:     3,
			wantErr: "error taking cards",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := drawStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			err = ManualDrawCards(&state, nil, testCase.player, testCase.count, testCase.rev)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("ManualDrawCards() error = %v; want containing %q", err, testCase.wantErr)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected ManualDrawCards mutated state")
			}
		})
	}
}

func drawStateForTest() model.MatchState {
	return model.MatchState{
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: []model.MatchCardID{"p1-hand-1"},
				Deck: []model.MatchCardID{"p1-deck-1", "p1-deck-2", "p1-deck-3"},
			},
			{
				ID:   "player-two",
				Hand: []model.MatchCardID{"p2-hand-1"},
				Deck: []model.MatchCardID{"p2-deck-1", "p2-deck-2"},
			},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    3,
		Turn: model.TurnState{
			Number:       2,
			ActivePlayer: "player-one",
			Phase:        model.PhaseMain,
		},
	}
}

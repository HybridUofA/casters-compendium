package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestPeekDeckTopsReturnsPrefixWithoutMutation(t *testing.T) {
	state := deckOpsStateForTest()
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	tops, err := PeekDeckTops(&state, "player-two", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tops, []model.MatchCardID{"p2-d1", "p2-d2"}) {
		t.Fatalf("tops = %#v", tops)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("PeekDeckTops mutated state")
	}
}

func TestMoveDeckTopToBottom(t *testing.T) {
	state := deckOpsStateForTest()
	if err := MoveDeckTopToBottom(&state, "player-one", "player-two", state.Revision); err != nil {
		t.Fatal(err)
	}
	want := []model.MatchCardID{"p2-d2", "p2-d3", "p2-d1"}
	if !reflect.DeepEqual(state.Players[1].Deck, want) {
		t.Fatalf("Deck = %#v; want %#v", state.Players[1].Deck, want)
	}
	if state.Revision != 2 {
		t.Fatalf("Revision = %d; want 2", state.Revision)
	}
}

func TestResolveDeckDigKeepsOneAndBottomsRest(t *testing.T) {
	state := deckOpsStateForTest()
	beforeHand := append([]model.MatchCardID(nil), state.Players[0].Hand...)
	if err := ResolveDeckDig(
		&state,
		nil,
		"player-one",
		"player-one",
		"p1-d2",
		[]model.MatchCardID{"p1-d3", "p1-d1"},
		state.Revision,
	); err != nil {
		t.Fatal(err)
	}
	wantHand := append(beforeHand, "p1-d2")
	if !reflect.DeepEqual(state.Players[0].Hand, wantHand) {
		t.Fatalf("Hand = %#v; want %#v", state.Players[0].Hand, wantHand)
	}
	wantDeck := []model.MatchCardID{"p1-d4", "p1-d3", "p1-d1"}
	if !reflect.DeepEqual(state.Players[0].Deck, wantDeck) {
		t.Fatalf("Deck = %#v; want %#v", state.Players[0].Deck, wantDeck)
	}
}

func TestResolveDeckDigRejectsStaleTops(t *testing.T) {
	state := deckOpsStateForTest()
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	err = ResolveDeckDig(
		&state,
		nil,
		"player-one",
		"player-one",
		"p1-d4",
		[]model.MatchCardID{"p1-d1", "p1-d2"},
		state.Revision,
	)
	if err == nil || !strings.Contains(err.Error(), "current top") {
		t.Fatalf("error = %v; want stale-top error", err)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected dig mutated state")
	}
}

func TestResolveDeckDigRejectsOpponentDeck(t *testing.T) {
	state := deckOpsStateForTest()
	err := ResolveDeckDig(
		&state,
		nil,
		"player-one",
		"player-two",
		"p2-d1",
		nil,
		state.Revision,
	)
	if err == nil || !strings.Contains(err.Error(), "own deck") {
		t.Fatalf("error = %v; want own-deck error", err)
	}
}

func deckOpsStateForTest() model.MatchState {
	return model.MatchState{
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: []model.MatchCardID{"p1-h1"},
				Deck: []model.MatchCardID{"p1-d1", "p1-d2", "p1-d3", "p1-d4"},
			},
			{
				ID:   "player-two",
				Deck: []model.MatchCardID{"p2-d1", "p2-d2", "p2-d3"},
			},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    1,
	}
}

package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestShufflePlayerDeckReordersAndBumpsRevision(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"a", "b", "c", "d", "e"}},
			{ID: "player-two", Deck: []model.MatchCardID{"x", "y"}},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    4,
	}
	beforeOpponent := append([]model.MatchCardID(nil), state.Players[1].Deck...)
	before := append([]model.MatchCardID(nil), state.Players[0].Deck...)

	// Same seed family as TestSeededRandomMakesShuffleRepeatable, which is known
	// not to leave a 5-ish card deck in its original order.
	if err := ShufflePlayerDeck(&state, NewSeededRandom(MatchSeed{First: 9876, Second: 5432}), "player-one", 4); err != nil {
		t.Fatalf("ShufflePlayerDeck() error = %v", err)
	}
	if state.Revision != 5 {
		t.Fatalf("Revision = %d; want 5", state.Revision)
	}
	if reflect.DeepEqual(state.Players[0].Deck, before) {
		t.Fatalf("deck was not shuffled: %v", state.Players[0].Deck)
	}
	if !reflect.DeepEqual(state.Players[1].Deck, beforeOpponent) {
		t.Fatal("shuffle mutated the other player's deck")
	}
	got := map[model.MatchCardID]int{}
	for _, id := range state.Players[0].Deck {
		got[id]++
	}
	for _, id := range before {
		if got[id] != 1 {
			t.Fatalf("shuffle lost or duplicated cards: before %v after %v", before, state.Players[0].Deck)
		}
	}
}

func TestShufflePlayerDeckRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	base := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"a", "b"}},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    1,
	}
	tests := []struct {
		name    string
		mutate  func(*model.MatchState)
		player  model.PlayerID
		rev     model.Revision
		random  RandomSource
		wantErr string
	}{
		{name: "stale revision", player: "player-one", rev: 0, random: NewSeededRandom(MatchSeed{First: 1, Second: 2}), wantErr: "expected revision 0"},
		{name: "nil random", player: "player-one", rev: 1, wantErr: "random source"},
		{
			name: "finished match",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusFinished
			},
			player:  "player-one",
			rev:     1,
			random:  NewSeededRandom(MatchSeed{First: 1, Second: 2}),
			wantErr: "not in progress",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := base
			state.Players[0].Deck = append([]model.MatchCardID(nil), base.Players[0].Deck...)
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			err = ShufflePlayerDeck(&state, testCase.random, testCase.player, testCase.rev)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v; want containing %q", err, testCase.wantErr)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected shuffle mutated state")
			}
		})
	}
}

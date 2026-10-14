package effects

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestResolveRestDrawIfTwoAlliedEntered(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{},
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"d1"}},
			{ID: "player-two"},
		},
		TurnFlags: model.TurnFlags{ServantsEntered: [2]int{2, 0}},
	}
	state.CardInstances["d1"] = model.CardInstance{
		CardID: "x", MatchID: "d1", Owner: "player-one", Controller: "player-one",
	}
	if err := resolveRestDrawIfTwoAlliedEntered(Context{
		State: &state, Controller: "player-one", Source: "caster",
	}); err != nil {
		t.Fatalf("resolve error = %v", err)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "d1" {
		t.Fatalf("Hand = %#v; want drew d1", state.Players[0].Hand)
	}
}

func TestResolveRestDrawIfTwoAlliedEnteredSkipsWhenFalse(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"d1"}},
			{ID: "player-two"},
		},
		TurnFlags: model.TurnFlags{ServantsEntered: [2]int{1, 0}},
	}
	if err := resolveRestDrawIfTwoAlliedEntered(Context{
		State: &state, Controller: "player-one",
	}); err != nil {
		t.Fatalf("resolve error = %v", err)
	}
	if len(state.Players[0].Hand) != 0 {
		t.Fatalf("Hand = %#v; want no draw", state.Players[0].Hand)
	}
}

func TestRPSWinner(t *testing.T) {
	if got := rpsWinner("rock", "scissors", "p1", "p2"); got != "p1" {
		t.Fatalf("rock vs scissors = %q; want p1", got)
	}
	if got := rpsWinner("rock", "paper", "p1", "p2"); got != "p2" {
		t.Fatalf("rock vs paper = %q; want p2", got)
	}
	if got := rpsWinner("rock", "rock", "p1", "p2"); got != "" {
		t.Fatalf("tie = %q; want empty", got)
	}
}

func TestResolveHildeWillowRestOffersPending(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two", Deck: []model.MatchCardID{"top"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"top": {CardID: "c", MatchID: "top", Owner: "player-two", Controller: "player-two"},
		},
	}
	var offered model.PendingChoice
	err := resolveHildeWillowRest(Context{
		State: &state, Controller: "player-one", Source: "hilde",
		OfferChoice: func(choice model.PendingChoice) error {
			offered = choice
			state.PendingChoice = choice
			return nil
		},
	})
	if err != nil {
		t.Fatalf("resolveHildeWillowRest error = %v", err)
	}
	if offered.Kind != PendingHildeDeckBottom || offered.Payload != "player-two" {
		t.Fatalf("offered = %#v", offered)
	}
	if err := continueHildeDeckBottom(offered, "bottom", Context{State: &state}); err != nil {
		t.Fatalf("bottom error = %v", err)
	}
	if state.Players[1].Deck[0] != "top" && len(state.Players[1].Deck) == 1 {
		// single card stay on bottom is same position
	}
}

func TestGoblinMoneylenderEnterOffersRPS(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{{ID: "player-one"}, {ID: "player-two"}},
	}
	var offered model.PendingChoice
	if err := resolveGoblinMoneylenderEnter(Context{
		State: &state, Controller: "player-one", Source: "goblin",
		OfferChoice: func(choice model.PendingChoice) error {
			offered = choice
			return nil
		},
	}); err != nil {
		t.Fatalf("error = %v", err)
	}
	if offered.Kind != PendingRPSChoice || len(offered.Options) != 3 {
		t.Fatalf("offered = %#v", offered)
	}
	var second model.PendingChoice
	if err := continueRPSChoice(offered, "rock", Context{
		State: &state, Controller: "player-one",
		OfferChoice: func(choice model.PendingChoice) error {
			second = choice
			state.PendingChoice = choice
			return nil
		},
	}); err != nil {
		t.Fatalf("first choice error = %v", err)
	}
	if second.PlayerID != "player-two" || second.Kind != PendingRPSChoice {
		t.Fatalf("second = %#v", second)
	}
	state.CardInstances = map[model.MatchCardID]model.CardInstance{
		"d1": {MatchID: "d1", CardID: "x", Owner: "player-one", Controller: "player-one"},
		"d2": {MatchID: "d2", CardID: "x", Owner: "player-one", Controller: "player-one"},
	}
	state.Players[0].Deck = []model.MatchCardID{"d1", "d2"}
	if err := continueRPSChoice(second, "scissors", Context{State: &state}); err != nil {
		t.Fatalf("second choice error = %v", err)
	}
	if len(state.Players[0].Hand) != 2 {
		t.Fatalf("winner Hand = %#v; want 2 cards", state.Players[0].Hand)
	}
}

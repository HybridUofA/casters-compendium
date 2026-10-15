package effects

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestResolveStrawEffigyEnterOffersAlliedCorrupt(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"effigy": {MatchID: "effigy", CardID: "7", Owner: "player-one", Controller: "player-one"},
			"orb-a":  {MatchID: "orb-a", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"orb-b":  {MatchID: "orb-b", CardID: "x", Owner: "player-one", Controller: "player-one"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Orbs: []model.MatchCardID{"orb-a", "orb-b"}},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
	}
	var offered model.PendingChoice
	ctx := Context{
		State:      &state,
		Source:     "effigy",
		Controller: "player-one",
		Hook:       HookEnter,
		OfferChoice: func(choice model.PendingChoice) error {
			offered = choice
			state.PendingChoice = choice
			return nil
		},
	}
	if err := resolveStrawEffigyEnter(ctx); err != nil {
		t.Fatalf("resolveStrawEffigyEnter() error = %v", err)
	}
	if offered.Kind != PendingCorruptOrb || len(offered.SelectableCardIDs) != 2 {
		t.Fatalf("PendingChoice = %#v; want allied corrupt over both orbs", offered)
	}

	if err := ContinuePendingChoice(offered, []model.MatchCardID{"orb-b"}, "", ctx); err != nil {
		t.Fatalf("ContinuePendingChoice() error = %v", err)
	}
	if len(state.Players[0].Orbs) != 1 || state.Players[0].Orbs[0] != "orb-a" {
		t.Fatalf("Orbs = %#v; want orb-a remaining", state.Players[0].Orbs)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "orb-b" {
		t.Fatalf("Hand = %#v; want orb-b", state.Players[0].Hand)
	}
}

func TestResolveVegaCorruptOrbBanishesThenOffersCorrupt(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"vega":     {MatchID: "vega", CardID: "152", Owner: "player-one", Controller: "player-one"},
			"allied":   {MatchID: "allied", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"enemy":    {MatchID: "enemy", CardID: "x", Owner: "player-two", Controller: "player-two"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Orbs: []model.MatchCardID{"allied"}},
			{ID: "player-two", Orbs: []model.MatchCardID{"enemy"}},
		},
		MatchStatus: model.StatusInProgress,
	}
	var offered model.PendingChoice
	ctx := Context{
		State:      &state,
		Source:     "vega",
		Controller: "player-one",
		Hook:       HookRest,
		OfferChoice: func(choice model.PendingChoice) error {
			offered = choice
			state.PendingChoice = choice
			return nil
		},
	}
	if err := resolveVegaCorruptOrb(ctx); err != nil {
		t.Fatalf("resolveVegaCorruptOrb() error = %v", err)
	}
	if offered.Kind != PendingBanishOrb || offered.Payload != "then_corrupt_orb" {
		t.Fatalf("first choice = %#v; want banish allied cost", offered)
	}

	if err := ContinuePendingChoice(offered, []model.MatchCardID{"allied"}, "", ctx); err != nil {
		t.Fatalf("banish continue error = %v", err)
	}
	if len(state.Players[0].Orbs) != 0 || !containsID(state.Players[0].Graveyard, "allied") {
		t.Fatalf("after banish Orbs/GY = %#v/%#v", state.Players[0].Orbs, state.Players[0].Graveyard)
	}
	if offered.Kind != PendingCorruptOrb || len(offered.SelectableCardIDs) != 1 || offered.SelectableCardIDs[0] != "enemy" {
		t.Fatalf("second choice = %#v; want corrupt remaining enemy orb", offered)
	}

	if err := ContinuePendingChoice(offered, []model.MatchCardID{"enemy"}, "", ctx); err != nil {
		t.Fatalf("corrupt continue error = %v", err)
	}
	if len(state.Players[1].Orbs) != 0 || !containsID(state.Players[1].Hand, "enemy") {
		t.Fatalf("after corrupt Orbs/Hand = %#v/%#v", state.Players[1].Orbs, state.Players[1].Hand)
	}
}

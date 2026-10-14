package effects

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestMasterEffectsIncludesCatalogAndAutomation(t *testing.T) {
	if len(MasterEffects) < 300 {
		t.Fatalf("MasterEffects len = %d; want full catalog inventory", len(MasterEffects))
	}
	handlers := DefaultHandlers()
	if len(handlers) < 100 {
		t.Fatalf("DefaultHandlers() len = %d; want large automation set", len(handlers))
	}
	var foundCashCat, foundInfini, foundTwinkle bool
	for _, h := range handlers {
		if h.Matches(HookEnter, "1281", "") {
			foundCashCat = true
		}
		if h.Matches(HookLastWords, "80", "") {
			foundInfini = true
		}
		if h.Matches(HookRest, "1228", "") {
			foundTwinkle = true
		}
	}
	if !foundCashCat || !foundInfini || !foundTwinkle {
		t.Fatalf("missing defaults: cashCat=%v infini=%v twinkle=%v", foundCashCat, foundInfini, foundTwinkle)
	}
}

func TestCashCatEnterDrawsEachPlayer(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"p1-d1"}},
			{ID: "player-two", Deck: []model.MatchCardID{"p2-d1"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-d1": {MatchID: "p1-d1", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"p2-d1": {MatchID: "p2-d1", CardID: "x", Owner: "player-two", Controller: "player-two"},
			"cat":   {MatchID: "cat", CardID: "1281", Owner: "player-one", Controller: "player-one"},
		},
	}
	if err := resolveCashCatEnter(Context{
		State: &state, Source: "cat", Controller: "player-one", Hook: HookEnter,
	}); err != nil {
		t.Fatalf("resolveCashCatEnter() error = %v", err)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "p1-d1" {
		t.Fatalf("P1 hand = %#v", state.Players[0].Hand)
	}
	if len(state.Players[1].Hand) != 1 || state.Players[1].Hand[0] != "p2-d1" {
		t.Fatalf("P2 hand = %#v", state.Players[1].Hand)
	}
}

func TestInfiniCoinLastWordsReturnsToHand(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Graveyard: []model.MatchCardID{"coin"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"coin": {MatchID: "coin", CardID: "80", Owner: "player-one", Controller: "player-one"},
		},
	}
	if err := resolveInfiniCoinLastWords(Context{
		State: &state, Source: "coin", Controller: "player-one", Hook: HookLastWords,
	}); err != nil {
		t.Fatalf("resolveInfiniCoinLastWords() error = %v", err)
	}
	if len(state.Players[0].Graveyard) != 0 || len(state.Players[0].Hand) != 1 {
		t.Fatalf("grave/hand = %#v / %#v", state.Players[0].Graveyard, state.Players[0].Hand)
	}
}

func TestTwinkleCharmRestDraws(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", Deck: []model.MatchCardID{"d1"}, CasterZone: []model.MatchCardID{"charm"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"d1":    {MatchID: "d1", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"charm": {MatchID: "charm", CardID: "1228", Owner: "player-one", Controller: "player-one"},
		},
	}
	if err := resolveTwinkleCharmRest(Context{
		State: &state, Source: "charm", Controller: "player-one", Hook: HookRest,
	}); err != nil {
		t.Fatalf("resolveTwinkleCharmRest() error = %v", err)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "d1" {
		t.Fatalf("Hand = %#v", state.Players[0].Hand)
	}
}

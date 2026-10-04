package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestBeginDrawOffersSageAdviceWhenBarrierControlled(t *testing.T) {
	state, catalog := sageAdviceStateForTest()
	if err := beginDrawOrOfferReplacement(&state, catalog, "player-one", 1); err != nil {
		t.Fatalf("beginDrawOrOfferReplacement() error = %v", err)
	}
	if state.PendingDraw.Step != model.PendingDrawOffer ||
		state.PendingDraw.PlayerID != "player-one" ||
		state.PendingDraw.Remaining != 1 {
		t.Fatalf("PendingDraw = %#v", state.PendingDraw)
	}
	if len(state.Players[0].Hand) != 0 {
		t.Fatal("offer must not draw before the player decides")
	}
}

func TestDeclineDrawReplacementDrawsNormally(t *testing.T) {
	state, catalog := sageAdviceStateForTest()
	if err := beginDrawOrOfferReplacement(&state, catalog, "player-one", 1); err != nil {
		t.Fatalf("beginDrawOrOfferReplacement() error = %v", err)
	}
	state.Revision = 5
	if err := DeclineDrawReplacement(&state, catalog, "player-one", 5); err != nil {
		t.Fatalf("DeclineDrawReplacement() error = %v", err)
	}
	if state.PendingDraw.Step != model.PendingDrawIdle {
		t.Fatalf("PendingDraw = %#v; want idle", state.PendingDraw)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "d1" {
		t.Fatalf("Hand = %#v; want [d1]", state.Players[0].Hand)
	}
}

func TestAcceptSageAdviceThenDigKeepsChosenCard(t *testing.T) {
	state, catalog := sageAdviceStateForTest()
	if err := beginDrawOrOfferReplacement(&state, catalog, "player-one", 1); err != nil {
		t.Fatalf("beginDrawOrOfferReplacement() error = %v", err)
	}
	state.Revision = 8
	if err := AcceptSageAdvice(&state, "player-one", 8); err != nil {
		t.Fatalf("AcceptSageAdvice() error = %v", err)
	}
	if state.PendingDraw.Step != model.PendingDrawDig {
		t.Fatalf("PendingDraw.Step = %q; want Dig", state.PendingDraw.Step)
	}
	if err := ResolveDeckDig(
		&state,
		catalog,
		"player-one",
		"player-one",
		"d2",
		[]model.MatchCardID{"d1", "d3"},
		state.Revision,
	); err != nil {
		t.Fatalf("ResolveDeckDig() error = %v", err)
	}
	if state.PendingDraw.Step != model.PendingDrawIdle {
		t.Fatalf("PendingDraw = %#v; want cleared after dig", state.PendingDraw)
	}
	if len(state.Players[0].Hand) != 1 || state.Players[0].Hand[0] != "d2" {
		t.Fatalf("Hand = %#v; want [d2]", state.Players[0].Hand)
	}
	if len(state.Players[0].Deck) < 2 ||
		state.Players[0].Deck[len(state.Players[0].Deck)-2] != "d1" ||
		state.Players[0].Deck[len(state.Players[0].Deck)-1] != "d3" {
		t.Fatalf("Deck bottom = %#v; want d1 then d3 at bottom", state.Players[0].Deck)
	}
}

func TestAcceptSageAdviceRejectsWrongPlayer(t *testing.T) {
	state, catalog := sageAdviceStateForTest()
	if err := beginDrawOrOfferReplacement(&state, catalog, "player-one", 1); err != nil {
		t.Fatalf("beginDrawOrOfferReplacement() error = %v", err)
	}
	state.Revision = 3
	err := AcceptSageAdvice(&state, "player-two", 3)
	if err == nil || !strings.Contains(err.Error(), "drawing player") {
		t.Fatalf("AcceptSageAdvice() error = %v; want drawing-player error", err)
	}
}

func sageAdviceStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"sage": {
				CardID: "sage-advice", MatchID: "sage", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
			"d1": {CardID: "c1", MatchID: "d1", Owner: "player-one", Controller: "player-one", CardCategory: model.CategoryPrintedCard},
			"d2": {CardID: "c2", MatchID: "d2", Owner: "player-one", Controller: "player-one", CardCategory: model.CategoryPrintedCard},
			"d3": {CardID: "c3", MatchID: "d3", Owner: "player-one", Controller: "player-one", CardCategory: model.CategoryPrintedCard},
			"d4": {CardID: "c4", MatchID: "d4", Owner: "player-one", Controller: "player-one", CardCategory: model.CategoryPrintedCard},
		},
		Players: [2]model.PlayerState{
			{
				ID:          "player-one",
				Deck:        []model.MatchCardID{"d1", "d2", "d3", "d4"},
				ServantZone: []model.MatchCardID{"sage"},
			},
			{ID: "player-two"},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusInProgress,
		Revision:    1,
		Turn:        model.TurnState{Number: 2, ActivePlayer: "player-one", Phase: model.PhaseDraw},
		NextLinkID:  1,
	}
	catalog := casterAetherCatalogForTest{
		"sage-advice": {
			ID: "sage-advice", Name: "Sage Advice", Type: "Barrier",
		},
		"c1": {ID: "c1", Name: "One", Type: "Servant"},
		"c2": {ID: "c2", Name: "Two", Type: "Servant"},
		"c3": {ID: "c3", Name: "Three", Type: "Servant"},
		"c4": {ID: "c4", Name: "Four", Type: "Servant"},
	}
	return state, catalog
}

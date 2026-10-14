package engine

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type effectCatalog map[string]gamecards.Card

func (c effectCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestCashCatEnterOnChaseResolve(t *testing.T) {
	t.Cleanup(ResetEffectHandlers)
	RegisterDefaultEffectHandlers()

	state := model.MatchState{
		Revision:             1,
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		PassCount:            1,
		Turn: model.TurnState{
			Number: 2, ActivePlayer: "player-one", Phase: model.PhaseMain,
		},
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Deck: []model.MatchCardID{"p1-draw"},
			},
			{
				ID:   "player-two",
				Deck: []model.MatchCardID{"p2-draw"},
			},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"cat": {
				MatchID: "cat", CardID: "1281", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
			"p1-draw": {MatchID: "p1-draw", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"p2-draw": {MatchID: "p2-draw", CardID: "x", Owner: "player-two", Controller: "player-two"},
		},
		ChaseLinks: []model.ChaseLink{{
			ID: 1, Kind: model.ChaseLinkCardPlay, Controller: "player-one",
			SourceCardID: "cat", EntryOrientation: model.OrientationRecovered,
		}},
	}
	catalog := effectCatalog{
		"1281": {ID: "1281", Name: "Cash Cat", Type: "Servant", Ability: "• Enter: Each player draws a card."},
	}
	if err := PassPriority(&state, catalog, "player-one", 1); err != nil {
		t.Fatalf("PassPriority() error = %v", err)
	}
	if !containsMatchID(state.Players[0].ServantZone, "cat") {
		t.Fatalf("ServantZone = %#v; Cash Cat should enter field", state.Players[0].ServantZone)
	}
	if !containsMatchID(state.Players[0].Hand, "p1-draw") || !containsMatchID(state.Players[1].Hand, "p2-draw") {
		t.Fatalf("hands after Enter = %#v / %#v", state.Players[0].Hand, state.Players[1].Hand)
	}
}

func TestInfiniCoinLastWordsOnDestroy(t *testing.T) {
	t.Cleanup(ResetEffectHandlers)
	RegisterDefaultEffectHandlers()

	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"coin"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"coin": {
				MatchID: "coin", CardID: "80", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
				Orientation: model.OrientationRecovered,
			},
		},
	}
	catalog := effectCatalog{"80": {ID: "80", Name: "Infini-Coin", Type: "Servant"}}
	if err := DestroyServant(&state, catalog, "coin"); err != nil {
		t.Fatalf("DestroyServant() error = %v", err)
	}
	if containsMatchID(state.Players[0].ServantZone, "coin") {
		t.Fatal("coin remained on field")
	}
	if containsMatchID(state.Players[0].Graveyard, "coin") {
		t.Fatal("coin stayed in graveyard after Last Words")
	}
	if !containsMatchID(state.Players[0].Hand, "coin") {
		t.Fatalf("Hand = %#v; want Infini-Coin returned", state.Players[0].Hand)
	}
}

func TestActivateTwinkleCharmRestDraws(t *testing.T) {
	t.Cleanup(ResetEffectHandlers)
	RegisterDefaultEffectHandlers()

	state := model.MatchState{
		Revision:             3,
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn:                 model.TurnState{ActivePlayer: "player-one", Phase: model.PhaseMain},
		Players: [2]model.PlayerState{
			{
				ID:         "player-one",
				Deck:       []model.MatchCardID{"draw-me"},
				CasterZone: []model.MatchCardID{"charm"},
			},
			{ID: "player-two"},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"charm": {
				MatchID: "charm", CardID: "1228", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
				Orientation: model.OrientationRecovered,
			},
			"draw-me": {MatchID: "draw-me", CardID: "x", Owner: "player-one", Controller: "player-one"},
		},
	}
	catalog := effectCatalog{
		"1228": {ID: "1228", Name: "Twinkle Charm", Type: "Caster", Ability: "• Rest: Draw a card."},
	}
	if err := ActivateRegisteredRest(&state, catalog, "player-one", "charm", 3); err != nil {
		t.Fatalf("ActivateRegisteredRest() error = %v", err)
	}
	if state.CardInstances["charm"].Orientation != model.OrientationRested {
		t.Fatal("charm should be rested")
	}
	if !containsMatchID(state.Players[0].Hand, "draw-me") {
		t.Fatalf("Hand = %#v", state.Players[0].Hand)
	}
	if state.Revision != 4 {
		t.Fatalf("Revision = %d; want 4", state.Revision)
	}
}

func containsMatchID(ids []model.MatchCardID, want model.MatchCardID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

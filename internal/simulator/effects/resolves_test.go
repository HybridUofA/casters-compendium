package effects

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type testCatalog map[string]gamecards.Card

func (c testCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestResolveBloomLadyAndConjureDraws(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: []model.MatchCardID{"h1", "h2"},
				Deck: []model.MatchCardID{"d1", "d2", "d3", "d4"},
			},
			{ID: "player-two"},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"h1":    {MatchID: "h1", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"h2":    {MatchID: "h2", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"d1":    {MatchID: "d1", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"d2":    {MatchID: "d2", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"d3":    {MatchID: "d3", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"d4":    {MatchID: "d4", CardID: "x", Owner: "player-one", Controller: "player-one"},
			"bloom": {MatchID: "bloom", CardID: "1263", Owner: "player-one", Controller: "player-one"},
		},
	}
	if err := resolveBloomLadyEnter(Context{
		State: &state, Source: "bloom", Controller: "player-one", Hook: HookEnter,
	}); err != nil {
		t.Fatalf("resolveBloomLadyEnter() error = %v", err)
	}
	if len(state.Players[0].Hand) != 2 {
		t.Fatalf("Hand after Bloom Lady = %#v; want 2 redrawn", state.Players[0].Hand)
	}
	if len(state.Players[0].Graveyard) != 2 {
		t.Fatalf("Graveyard = %#v; want discarded hand", state.Players[0].Graveyard)
	}

	state.Players[0].Deck = append(state.Players[0].Deck, "d5", "d6", "d7")
	state.CardInstances["d5"] = model.CardInstance{MatchID: "d5", CardID: "x", Owner: "player-one", Controller: "player-one"}
	state.CardInstances["d6"] = model.CardInstance{MatchID: "d6", CardID: "x", Owner: "player-one", Controller: "player-one"}
	state.CardInstances["d7"] = model.CardInstance{MatchID: "d7", CardID: "x", Owner: "player-one", Controller: "player-one"}
	before := len(state.Players[0].Hand)
	if err := resolveDraw3(Context{
		State: &state, Controller: "player-one", Hook: HookResolve,
	}); err != nil {
		t.Fatalf("resolveDraw3() error = %v", err)
	}
	if len(state.Players[0].Hand) != before+3 {
		t.Fatalf("Hand after draw3 = %#v", state.Players[0].Hand)
	}
}

func TestResolveDestroyAllAndReturnAll(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"s1"}},
			{ID: "player-two", ServantZone: []model.MatchCardID{"s2"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"s1": {
				MatchID: "s1", CardID: "cheap", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
			"s2": {
				MatchID: "s2", CardID: "expensive", Owner: "player-two", Controller: "player-two",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
		},
	}
	catalog := testCatalog{
		"cheap":     {ID: "cheap", Type: "Servant", CostLevel: "2"},
		"expensive": {ID: "expensive", Type: "Servant", CostLevel: "6"},
	}
	destroyed := []model.MatchCardID{}
	ctx := Context{
		State: &state, Controller: "player-one", Catalog: catalog,
		DestroyField: func(id model.MatchCardID) error {
			destroyed = append(destroyed, id)
			// mimic destroy to GY
			return opsDestroyForTest(&state, id)
		},
	}
	if err := resolveDestroyAllServantsCostAtMost2(ctx); err != nil {
		t.Fatalf("destroy all ≤2: %v", err)
	}
	if len(destroyed) != 1 || destroyed[0] != "s1" {
		t.Fatalf("destroyed = %#v; want only s1", destroyed)
	}

	state = model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"s1"}},
			{ID: "player-two", ServantZone: []model.MatchCardID{"s2"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"s1": {
				MatchID: "s1", CardID: "cheap", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
			"s2": {
				MatchID: "s2", CardID: "expensive", Owner: "player-two", Controller: "player-two",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
		},
	}
	if err := resolveReturnAllServantsToHand(Context{
		State: &state, Controller: "player-one", Catalog: catalog,
	}); err != nil {
		t.Fatalf("return all: %v", err)
	}
	if len(state.Players[0].ServantZone) != 0 || len(state.Players[1].ServantZone) != 0 {
		t.Fatalf("servants remain on field")
	}
}

func opsDestroyForTest(state *model.MatchState, id model.MatchCardID) error {
	inst := state.CardInstances[id]
	for pi := range state.Players {
		zone := &state.Players[pi].ServantZone
		for i, card := range *zone {
			if card == id {
				*zone = append((*zone)[:i], (*zone)[i+1:]...)
				state.Players[pi].Graveyard = append(state.Players[pi].Graveyard, id)
				inst.Controller = inst.Owner
				state.CardInstances[id] = inst
				return nil
			}
		}
	}
	return nil
}

func TestOfferDestroyEnemyOpensPendingChoice(t *testing.T) {
	state := model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two", ServantZone: []model.MatchCardID{"enemy"}},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"enemy": {
				MatchID: "enemy", CardID: "e", Owner: "player-two", Controller: "player-two",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
			},
			"mantis": {MatchID: "mantis", CardID: "30", Owner: "player-one", Controller: "player-one"},
		},
	}
	var offered model.PendingChoice
	err := resolveDestroyEnemyServantCostAtMost3(Context{
		State: &state, Source: "mantis", Controller: "player-one",
		Catalog: testCatalog{"e": {ID: "e", Type: "Servant", CostLevel: "2"}},
		OfferChoice: func(choice model.PendingChoice) error {
			offered = choice
			return nil
		},
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if offered.Kind != PendingDestroyEnemyServant || len(offered.SelectableCardIDs) != 1 {
		t.Fatalf("offered = %#v", offered)
	}
}

func TestDefaultHandlersCountGrew(t *testing.T) {
	handlers := DefaultHandlers()
	if len(handlers) < 100 {
		t.Fatalf("DefaultHandlers() = %d; want large automation set", len(handlers))
	}
}

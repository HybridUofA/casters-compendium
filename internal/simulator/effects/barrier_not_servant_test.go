package effects

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type barrierFXCatalog map[string]gamecards.Card

func (c barrierFXCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestZoneServantsExcludesBarriers(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant": {MatchID: "servant", CardID: "s", Owner: "player-one", Controller: "player-one"},
			"barrier": {MatchID: "barrier", CardID: "b", Owner: "player-one", Controller: "player-one"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"servant", "barrier"}},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
	}
	ctx := Context{
		State:      &state,
		Controller: "player-one",
		Catalog: barrierFXCatalog{
			"s": {ID: "s", Type: "Servant"},
			"b": {ID: "b", Type: "Barrier"},
		},
	}
	got := zoneServants(ctx, 0)
	if len(got) != 1 || got[0] != "servant" {
		t.Fatalf("zoneServants = %#v; want [servant]", got)
	}
	if countZoneServants(ctx, 0) != 1 {
		t.Fatalf("countZoneServants = %d; want 1", countZoneServants(ctx, 0))
	}
}

func TestAbolitionLeavesBarriersOnField(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant": {MatchID: "servant", CardID: "s", Owner: "player-one", Controller: "player-one"},
			"barrier": {MatchID: "barrier", CardID: "b", Owner: "player-one", Controller: "player-one"},
		},
		Players: [2]model.PlayerState{
			{
				ID: "player-one",
				ServantZone: []model.MatchCardID{"servant", "barrier"},
				Deck:        []model.MatchCardID{},
			},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
	}
	ctx := Context{
		State:      &state,
		Controller: "player-one",
		Catalog: barrierFXCatalog{
			"s": {ID: "s", Type: "Servant"},
			"b": {ID: "b", Type: "Barrier"},
		},
	}
	if err := resolveAbolition(ctx); err != nil {
		t.Fatalf("resolveAbolition() error = %v", err)
	}
	if len(state.Players[0].ServantZone) != 1 || state.Players[0].ServantZone[0] != "barrier" {
		t.Fatalf("ServantZone = %#v; want barrier remaining", state.Players[0].ServantZone)
	}
	if !containsID(state.Players[0].Deck, "servant") {
		t.Fatalf("Deck = %#v; want servant on bottom", state.Players[0].Deck)
	}
}

func TestCharylThomasComparesServantCountsNotBarriers(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"ally":    {MatchID: "ally", CardID: "s", Owner: "player-one", Controller: "player-one"},
			"barrier": {MatchID: "barrier", CardID: "b", Owner: "player-one", Controller: "player-one"},
			"enemy":   {MatchID: "enemy", CardID: "s", Owner: "player-two", Controller: "player-two"},
			"enemy2":  {MatchID: "enemy2", CardID: "s", Owner: "player-two", Controller: "player-two"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"ally", "barrier"}},
			{ID: "player-two", ServantZone: []model.MatchCardID{"enemy", "enemy2"}},
		},
		MatchStatus: model.StatusInProgress,
	}
	ctx := Context{
		State:      &state,
		Controller: "player-one",
		Catalog: barrierFXCatalog{
			"s": {ID: "s", Type: "Servant"},
			"b": {ID: "b", Type: "Barrier"},
		},
	}
	// 1 servant vs 2 servants → Charyl should arm; zone lengths are equal (2 vs 2) if barriers counted.
	if err := resolveCharylThomasRest(ctx); err != nil {
		t.Fatalf("resolveCharylThomasRest() error = %v", err)
	}
	if !state.ContinuousUntilEOT.RestedAlliedCannotBeAttacked[0] {
		t.Fatal("Charyl should arm when servant count is lower, ignoring barriers")
	}
}

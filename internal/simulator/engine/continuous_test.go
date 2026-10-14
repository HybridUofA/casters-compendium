package engine

import (
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type continuousCatalog map[string]gamecards.Card

func (c continuousCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestSlowStartBlocksAttackOnEnterTurn(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"attacker": {
				CardID: "bear", MatchID: "attacker", Controller: "player-one",
				Orientation: model.OrientationRecovered, EnteredOnTurn: 2,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"attacker"}},
			{ID: "player-two"},
		},
		Turn: model.TurnState{Number: 2, ActivePlayer: "player-one", Phase: model.PhaseBattle},
	}
	catalog := continuousCatalog{
		"bear": {ID: "bear", Ability: "• Slow Start (I cannot attack the turn I am played to the field.)"},
	}
	err := validateAttackerContinuous(&state, catalog, "attacker")
	if err == nil || !strings.Contains(err.Error(), "Slow Start") {
		t.Fatalf("error = %v; want Slow Start", err)
	}
	state.Turn.Number = 3
	if err := validateAttackerContinuous(&state, catalog, "attacker"); err != nil {
		t.Fatalf("later turn should allow attack: %v", err)
	}
}

func TestRestedAlliedCannotBeAttackedFromBarrier(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"barrier": {CardID: "27", MatchID: "barrier", Controller: "player-two"},
			"target": {
				CardID: "t", MatchID: "target", Controller: "player-two",
				Orientation: model.OrientationRested,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two", ServantZone: []model.MatchCardID{"barrier", "target"}},
		},
	}
	catalog := continuousCatalog{
		"27": {ID: "27", Ability: "• Rested allied servants can't be attacked."},
		"t":  {ID: "t", Ability: ""},
	}
	if !servantCannotBeAttacked(&state, catalog, "target", 1) {
		t.Fatal("rested allied servant should be protected by Barrier of Arrogance")
	}
}

func TestPlayerCannotBeAttackedUntilEOT(t *testing.T) {
	state := model.MatchState{}
	state.ContinuousUntilEOT.PlayerCannotBeAttacked[1] = true
	if !playerCannotBeAttacked(&state, 1) {
		t.Fatal("expected player protection")
	}
}

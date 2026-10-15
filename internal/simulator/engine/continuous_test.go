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

func TestSophomoreSlumpGrantsGlobalSlowStart(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"slump": {CardID: "160", MatchID: "slump", Controller: "player-one"},
			"attacker": {
				CardID: "bear", MatchID: "attacker", Controller: "player-two",
				Orientation: model.OrientationRecovered, EnteredOnTurn: 2,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"slump"}},
			{ID: "player-two", ServantZone: []model.MatchCardID{"attacker"}},
		},
		Turn: model.TurnState{Number: 2},
	}
	catalog := continuousCatalog{
		"160":  {ID: "160", Ability: "• All servants gain Slow Start. (I cannot attack the turn I am played to the field.)"},
		"bear": {ID: "bear", Ability: ""},
	}
	if !hasSlowStartThisTurn(&state, catalog, "attacker") {
		t.Fatal("Sophomore Slump should grant Slow Start on enter turn")
	}
}

func TestWorldOfRageForcesEnterRecovered(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"rage":   {CardID: "151", MatchID: "rage", Controller: "player-one"},
			"source": {CardID: "bear", MatchID: "source", Owner: "player-two", Controller: "player-two"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", ServantZone: []model.MatchCardID{"rage"}},
			{ID: "player-two"},
		},
	}
	catalog := continuousCatalog{
		"151": {
			ID: "151",
			Ability: "• Enemy servants enter the field recovered.\n• Enemy servants cannot be switched except by spell or ability.",
		},
		"bear": {ID: "bear", Type: "Servant"},
	}
	link := model.ChaseLink{
		Controller: "player-two", SourceCardID: "source",
		Kind: model.ChaseLinkCardPlay, EntryOrientation: model.OrientationReversed,
	}
	if err := resolveServantCardPlay(&state, catalog, link, state.CardInstances["source"]); err != nil {
		t.Fatalf("resolveServantCardPlay: %v", err)
	}
	if state.CardInstances["source"].Orientation != model.OrientationRecovered {
		t.Fatalf("orientation = %q; want Recovered under World of Rage", state.CardInstances["source"].Orientation)
	}
}

func TestCardsLoseBreakUnderSaaya(t *testing.T) {
	orbs := make([]model.MatchCardID, 6)
	instances := map[model.MatchCardID]model.CardInstance{
		"saaya": {CardID: "186", MatchID: "saaya", Controller: "player-one"},
	}
	for i := range orbs {
		id := model.MatchCardID(string(rune('a' + i)))
		orbs[i] = id
		instances[id] = model.CardInstance{CardID: "orb", MatchID: id, Controller: "player-two"}
	}
	state := model.MatchState{
		CardInstances: instances,
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"saaya"}},
			{ID: "player-two", Orbs: orbs},
		},
	}
	catalog := continuousCatalog{
		"186": {
			ID: "186",
			Ability: "• As long as your opponent controls six or more orbs, cards they control lose Break.\n• Rest: Target allied servant with cost of 2 or less gains +500 ATK until end of turn.",
		},
		"orb": {ID: "orb"},
	}
	if !cardsLoseBreak(&state, catalog, 1) {
		t.Fatal("expected cardsLoseBreak when opponent has Saaya and 6 orbs")
	}
	state.Players[1].Orbs = state.Players[1].Orbs[:5]
	if cardsLoseBreak(&state, catalog, 1) {
		t.Fatal("should not lose Break with fewer than 6 orbs")
	}
}

func TestRestedAlliedCannotBeAttackedWhileDelphiniumReversed(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"delphinium": {
				CardID: "245", MatchID: "delphinium", Controller: "player-two",
				Orientation: model.OrientationRecovered,
			},
			"target": {
				CardID: "t", MatchID: "target", Controller: "player-two",
				Orientation: model.OrientationRested,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two", ServantZone: []model.MatchCardID{"delphinium", "target"}},
		},
	}
	catalog := continuousCatalog{
		"245": {
			ID: "245",
			Ability: "• Your opponent cannot attack allied rested servants while I am reversed.",
		},
		"t": {ID: "t", Ability: ""},
	}
	if servantCannotBeAttacked(&state, catalog, "target", 1) {
		t.Fatal("rested ally must remain attackable while Delphinium is Recovered")
	}
	delphinium := state.CardInstances["delphinium"]
	delphinium.Orientation = model.OrientationReversed
	state.CardInstances["delphinium"] = delphinium
	if !servantCannotBeAttacked(&state, catalog, "target", 1) {
		t.Fatal("rested ally must be protected while Delphinium is Reversed")
	}
}

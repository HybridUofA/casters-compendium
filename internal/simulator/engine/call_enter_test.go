package engine

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type callEnterCatalog map[string]gamecards.Card

func (c callEnterCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestCallFaceUpMerlinOpensPeekOrbPendingChoice(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	t.Cleanup(ResetEffectHandlers)

	state := callStateForTest()
	// Replace hand card with Merlin printed id 176.
	merlinID := model.MatchCardID("p1-hand-b")
	inst := state.CardInstances[merlinID]
	inst.CardID = "176"
	state.CardInstances[merlinID] = inst
	state.CardInstances["orb-1"] = model.CardInstance{
		MatchID: "orb-1", CardID: "orb-def", Owner: "player-two", Controller: "player-two",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.Players[1].Orbs = []model.MatchCardID{"orb-1"}

	catalog := callEnterCatalog{
		"176": {
			ID: "176", Name: "Merlin", Type: "Caster", CostLevel: "1",
			Ability: "• Enter : Look at target enemy orb, then return it. Draw a card.",
		},
		"orb-def": {ID: "orb-def", Type: "Servant"},
	}
	// Other hand/caster cards still need definitions for validation.
	for id, instance := range state.CardInstances {
		if _, ok := catalog[string(instance.CardID)]; ok {
			continue
		}
		catalog[string(instance.CardID)] = gamecards.Card{
			ID: string(instance.CardID), Type: "Caster", CostLevel: "1", Name: string(id),
		}
	}

	if err := CallFaceUpLevelOne(&state, catalog, "player-one", merlinID, state.Revision); err != nil {
		t.Fatalf("CallFaceUpLevelOne() error = %v", err)
	}
	if state.PendingChoice.PlayerID != "player-one" || state.PendingChoice.Kind != effects.PendingPeekOrb {
		t.Fatalf("PendingChoice = %#v; want Merlin Enter peek-orb prompt", state.PendingChoice)
	}
}

func TestIntelligentStaplerEnterOffersEnemyBanishAfterTwoPicks(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	t.Cleanup(ResetEffectHandlers)

	state := battleStateForTest()
	state.CardInstances["stapler"] = model.CardInstance{
		MatchID: "stapler", CardID: "53", Owner: "player-two", Controller: "player-two",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
		Orientation: model.OrientationRecovered,
	}
	state.Players[1].ServantZone = []model.MatchCardID{"stapler"}
	catalog := callEnterCatalog{
		"53": {
			ID: "53", Name: "Intelligent Stapler", Type: "Servant", CostLevel: "4",
			Ability: "• Enter, Choose two target servants an enemy player controls: That enemy player banishes one of them.",
		},
		"definition-p1-attacker": {
			ID: "definition-p1-attacker", Type: "Servant", CostLevel: "4",
		},
		"definition-p2-defender": {
			ID: "definition-p2-defender", Type: "Servant", CostLevel: "3",
		},
	}
	// Ensure player-one has two servants to choose.
	state.CardInstances["p1-cookie"] = battleCard("p1-cookie", "player-one", model.OrientationRecovered)
	cookie := state.CardInstances["p1-cookie"]
	cookie.CardID = "definition-p1-attacker"
	state.CardInstances["p1-cookie"] = cookie
	state.Players[0].ServantZone = []model.MatchCardID{"p1-attacker", "p1-cookie"}

	if err := triggerEnterEffects(&state, catalog, "stapler"); err != nil {
		t.Fatalf("triggerEnterEffects() error = %v", err)
	}
	if state.PendingChoice.Kind != effects.PendingStaplerPick {
		t.Fatalf("PendingChoice = %#v; want stapler pick", state.PendingChoice)
	}
	first := state.PendingChoice.SelectableCardIDs[0]
	if err := ResolvePendingChoice(&state, catalog, "player-two", "", []model.MatchCardID{first}, state.Revision); err != nil {
		t.Fatalf("ResolvePendingChoice(first) error = %v", err)
	}
	if state.PendingChoice.Kind != effects.PendingStaplerPick {
		t.Fatalf("PendingChoice after first = %#v; want second pick", state.PendingChoice)
	}
	second := state.PendingChoice.SelectableCardIDs[0]
	if err := ResolvePendingChoice(&state, catalog, "player-two", "", []model.MatchCardID{second}, state.Revision); err != nil {
		t.Fatalf("ResolvePendingChoice(second) error = %v", err)
	}
	if state.PendingChoice.PlayerID != "player-one" ||
		state.PendingChoice.Kind != effects.PendingBanishEnemyServant ||
		len(state.PendingChoice.SelectableCardIDs) != 2 {
		t.Fatalf("PendingChoice = %#v; want enemy banish of two chosen servants", state.PendingChoice)
	}
}

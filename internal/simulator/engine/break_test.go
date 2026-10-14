package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestCorruptOrbOffersBreakWhenCardDeclaresIt(t *testing.T) {
	state, catalog := breakBattleStateForTest()
	state.Players[1].ServantZone = nil

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}
	if state.PendingBreak.PlayerID != "player-two" ||
		len(state.PendingBreak.CardIDs) != 1 ||
		state.PendingBreak.CardIDs[0] != "p2-break-orb" {
		t.Fatalf("PendingBreak = %#v", state.PendingBreak)
	}
	if state.PrioritySequenceOpen {
		t.Fatal("priority reopened before Break decision")
	}
	if !slicesContains(state.Players[1].Hand, "p2-break-orb") {
		t.Fatal("corrupted Break card did not enter hand")
	}
}

func TestDeclineBreakReopensPriorityForActivePlayer(t *testing.T) {
	state, catalog := breakBattleStateForTest()
	state.Players[1].ServantZone = nil
	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)
	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}

	if err := DeclineBreak(&state, "player-two", state.Revision); err != nil {
		t.Fatalf("DeclineBreak() error = %v", err)
	}
	if state.PendingBreak.PlayerID != "" || len(state.PendingBreak.CardIDs) != 0 {
		t.Fatalf("PendingBreak = %#v; want cleared", state.PendingBreak)
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != "player-one" {
		t.Fatalf("priority = open %t holder %q; want active player", state.PrioritySequenceOpen, state.PriorityHolder)
	}
	if !slicesContains(state.Players[1].Hand, "p2-break-orb") {
		t.Fatal("declined Break removed the card from hand")
	}
}

func TestPlayBreakPutsCardOnChaseWithoutCost(t *testing.T) {
	state, catalog := breakBattleStateForTest()
	state.Players[1].ServantZone = nil
	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)
	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}

	if err := PlayBreak(&state, catalog, "player-two", "p2-break-orb", model.OrientationRecovered, state.Revision); err != nil {
		t.Fatalf("PlayBreak() error = %v", err)
	}
	if slicesContains(state.Players[1].Hand, "p2-break-orb") {
		t.Fatal("played Break card remained in hand")
	}
	if len(state.ChaseLinks) != 1 || state.ChaseLinks[0].SourceCardID != "p2-break-orb" {
		t.Fatalf("ChaseLinks = %#v", state.ChaseLinks)
	}
	if state.PendingBreak.PlayerID != "" || len(state.PendingBreak.CardIDs) != 0 {
		t.Fatalf("PendingBreak = %#v; want cleared after play", state.PendingBreak)
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != "player-two" {
		t.Fatalf("priority = open %t holder %q; want break player", state.PrioritySequenceOpen, state.PriorityHolder)
	}
}

func TestDeclineBreakRejectsWrongPlayer(t *testing.T) {
	state, catalog := breakBattleStateForTest()
	state.Players[1].ServantZone = nil
	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)
	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}
	err := DeclineBreak(&state, "player-one", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "owner") {
		t.Fatalf("DeclineBreak() error = %v; want owner error", err)
	}
}

func TestCardDeclaresBreakRecognizesLabels(t *testing.T) {
	if !cardDeclaresBreak("• Break (When this card is put into your hand from an orb zone, you may play it immediately without paying its cost.)") {
		t.Fatal("expected parenthetical Break label")
	}
	if !cardDeclaresBreak("• [Break]") {
		t.Fatal("expected bracketed Break label")
	}
	if cardDeclaresBreak("• As long as your opponent controls six or more orbs, cards they control lose Break.") {
		t.Fatal("effect prose must not count as declaring Break")
	}
}

func TestPlayBreakFirewheelEnterOffersDestroyTarget(t *testing.T) {
	ResetEffectHandlers()
	RegisterDefaultEffectHandlers()
	t.Cleanup(ResetEffectHandlers)

	state, catalog := firewheelBreakStateForTest()
	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)
	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}
	if err := PlayBreak(&state, catalog, "player-two", "p2-firewheel", model.OrientationRecovered, state.Revision); err != nil {
		t.Fatalf("PlayBreak() error = %v", err)
	}
	// Both players pass so Break resolves from Chase and Enter fires.
	if err := PassPriority(&state, catalog, "player-two", state.Revision); err != nil {
		t.Fatalf("PassPriority(break player) error = %v", err)
	}
	if err := PassPriority(&state, catalog, "player-one", state.Revision); err != nil {
		t.Fatalf("PassPriority(active) error = %v", err)
	}
	if state.PendingChoice.PlayerID != "player-two" ||
		state.PendingChoice.Kind != "fx:destroy_enemy_servant" ||
		len(state.PendingChoice.SelectableCardIDs) == 0 {
		t.Fatalf("PendingChoice = %#v; want Firewheel Enter destroy prompt", state.PendingChoice)
	}
	target := state.PendingChoice.SelectableCardIDs[0]
	if err := ResolvePendingChoice(&state, catalog, "player-two", "", []model.MatchCardID{target}, state.Revision); err != nil {
		t.Fatalf("ResolvePendingChoice() error = %v", err)
	}
	if slicesContains(state.Players[0].ServantZone, target) {
		t.Fatalf("target %q remained on the field after Firewheel Enter", target)
	}
}

func firewheelBreakStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := battleStateForTest()
	// Cheap enemy servant (Crispy Cookie cost 1) that Firewheel Enter can destroy.
	state.CardInstances["p1-cookie"] = battleCard("p1-cookie", "player-one", model.OrientationRecovered)
	state.Players[0].ServantZone = []model.MatchCardID{"p1-attacker", "p1-cookie"}
	state.CardInstances["p2-firewheel"] = battleCard("p2-firewheel", "player-two", model.OrientationRecovered)
	state.Players[1].Orbs = []model.MatchCardID{"p2-firewheel", "p2-orb-2"}
	state.Players[1].ServantZone = nil
	catalog := casterAetherCatalogForTest{
		"definition-p1-attacker": {
			ID: "definition-p1-attacker", Type: "Servant", Attack: "3000", Defense: "2000", CostLevel: "4",
		},
		"definition-p1-cookie": {
			ID: "definition-p1-cookie", Type: "Servant", Attack: "1000", Defense: "1000", CostLevel: "1",
		},
		"definition-p2-defender": {
			ID: "definition-p2-defender", Type: "Servant", Attack: "2000", Defense: "1000",
		},
		// Printed Firewheel id so HookEnter master recipe matches.
		"12": {
			ID:        "12",
			Name:      "Firewheel",
			Type:      "Servant",
			Element:   "Ignus",
			CostLevel: "3",
			Attack:    "2000",
			Defense:   "1000",
			Ability:   "• Break (When this card is put into your hand from an orb zone, you may play it immediately without paying its cost.)\n• Enter: Destroy target enemy servant with a cost of 3 or less.",
		},
		"definition-p2-orb-2": {
			ID: "definition-p2-orb-2", Type: "Servant", Ability: "• Enter: Draw a card.",
		},
	}
	inst := state.CardInstances["p2-firewheel"]
	inst.CardID = "12"
	state.CardInstances["p2-firewheel"] = inst
	cookie := state.CardInstances["p1-cookie"]
	cookie.CardID = "definition-p1-cookie"
	state.CardInstances["p1-cookie"] = cookie
	return state, catalog
}

func breakBattleStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := battleStateForTest()
	state.CardInstances["p2-break-orb"] = battleCard("p2-break-orb", "player-two", model.OrientationRecovered)
	state.Players[1].Orbs = []model.MatchCardID{"p2-break-orb", "p2-orb-2"}
	catalog := casterAetherCatalogForTest{
		"definition-p1-attacker": {
			ID: "definition-p1-attacker", Type: "Servant", Attack: "3000", Defense: "2000",
		},
		"definition-p2-defender": {
			ID: "definition-p2-defender", Type: "Servant", Attack: "2000", Defense: "1000",
		},
		"definition-p2-break-orb": {
			ID:        "definition-p2-break-orb",
			Type:      "Servant",
			Element:   "Ignus",
			CostLevel: "3",
			Attack:    "2000",
			Defense:   "1000",
			Ability:   "• Break (When this card is put into your hand from an orb zone, you may play it immediately without paying its cost.)",
		},
		"definition-p2-orb-2": {
			ID: "definition-p2-orb-2", Type: "Servant", Ability: "• Enter: Draw a card.",
		},
	}
	return state, catalog
}

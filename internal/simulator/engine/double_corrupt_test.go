package engine

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestDoubleCorruptRequiresTwoOrbChoices(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil
	attacker := state.CardInstances["p1-attacker"]
	attacker.CardID = "definition-double-corrupt"
	state.CardInstances["p1-attacker"] = attacker
	catalog["definition-double-corrupt"] = gamecards.Card{
		ID: "definition-double-corrupt", Type: "Servant", Attack: "3000", Defense: "2000",
		Ability: "• Double Corrupt",
	}

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if state.Attack.CorruptCount != 2 {
		t.Fatalf("CorruptCount = %d; want 2", state.Attack.CorruptCount)
	}
	if err := CorruptOrb(&state, catalog, "player-one", 0, state.Revision); err == nil ||
		!strings.Contains(err.Error(), "exactly 2") {
		t.Fatalf("CorruptOrb() error = %v; want exactly-2 requirement", err)
	}
	if err := CorruptOrbs(&state, catalog, "player-one", []int{0, 1}, state.Revision); err != nil {
		t.Fatalf("CorruptOrbs() error = %v", err)
	}
	if len(state.Players[1].Orbs) != 0 {
		t.Fatalf("Orbs = %#v; want both corrupted", state.Players[1].Orbs)
	}
	if !reflect.DeepEqual(state.Players[1].Hand, []model.MatchCardID{"p2-orb-1", "p2-orb-2"}) {
		t.Fatalf("Hand = %#v; want both corrupted orbs in selection order", state.Players[1].Hand)
	}
	if state.MatchStatus != model.StatusInProgress {
		t.Fatal("Double Corrupt must not finish the match by emptying Orbs")
	}
}

func TestDoubleCorruptAgainstOneOrbDoesNotWin(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil
	state.Players[1].Orbs = []model.MatchCardID{"p2-orb-1"}
	attacker := state.CardInstances["p1-attacker"]
	attacker.GrantedDoubleCorrupt = true
	state.CardInstances["p1-attacker"] = attacker

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if state.Attack.CorruptCount != 1 {
		t.Fatalf("CorruptCount = %d; want 1 when only one Orb remains", state.Attack.CorruptCount)
	}
	if err := CorruptOrbs(&state, catalog, "player-one", []int{0}, state.Revision); err != nil {
		t.Fatalf("CorruptOrbs() error = %v", err)
	}
	if state.MatchStatus != model.StatusInProgress {
		t.Fatalf("MatchStatus = %q; Double Corrupt vs one Orb must not win", state.MatchStatus)
	}
	if len(state.Players[1].Orbs) != 0 || !slicesContains(state.Players[1].Hand, "p2-orb-1") {
		t.Fatalf("Orbs/Hand = %#v/%#v", state.Players[1].Orbs, state.Players[1].Hand)
	}
}

func TestSetGrantedDoubleCorruptEnablesDoubleCorrupt(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	if err := SetGrantedDoubleCorrupt(&state, "player-one", "p1-attacker", true, state.Revision); err != nil {
		t.Fatalf("SetGrantedDoubleCorrupt() error = %v", err)
	}
	if !state.CardInstances["p1-attacker"].GrantedDoubleCorrupt {
		t.Fatal("granted marker not set")
	}
	if !hasDoubleCorrupt(&state, catalog, "p1-attacker") {
		t.Fatal("granted marker should count as Double Corrupt")
	}
}

func TestCompleteBattlePhaseRejectsMandatoryEligibleAttacker(t *testing.T) {
	state := battleStateForTest()
	state.PrioritySequenceOpen = false
	state.PriorityHolder = ""
	state.PassCount = 0
	before := state.Revision

	err := completeBattlePhase(&state, nil, "player-one")
	if err == nil || !strings.Contains(err.Error(), "able to attack") {
		t.Fatalf("completeBattlePhase() error = %v; want mandatory-attack error", err)
	}
	if state.Turn.Phase != model.PhaseBattle || state.Revision != before {
		t.Fatal("rejected battle completion mutated phase/revision")
	}

	for _, cardID := range state.Players[0].ServantZone {
		instance := state.CardInstances[cardID]
		instance.Orientation = model.OrientationRested
		state.CardInstances[cardID] = instance
	}
	if err := completeBattlePhase(&state, nil, "player-one"); err != nil {
		t.Fatalf("completeBattlePhase() after resting attackers error = %v", err)
	}
	if state.Turn.Phase != model.PhaseEnd {
		t.Fatalf("phase = %q; want End", state.Turn.Phase)
	}
}

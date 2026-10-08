package engine

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestDeclareAttackRestsAttackerAndRecordsPlayerTarget(t *testing.T) {
	state := battleStateForTest()

	err := DeclareAttack(
		&state,
		nil,
		"player-one",
		"p1-attacker",
		model.AttackTargetPlayer,
		"",
		state.Revision,
	)
	if err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	if state.CardInstances["p1-attacker"].Orientation != model.OrientationRested {
		t.Fatalf("attacker orientation = %q; want Rested", state.CardInstances["p1-attacker"].Orientation)
	}
	wantAttack := model.AttackState{
		AttackerID:   "p1-attacker",
		TargetKind:   model.AttackTargetPlayer,
		TargetCardID: "",
		Step:         model.BattleStepDeclared,
	}
	if state.Attack != wantAttack {
		t.Fatalf("Attack = %#v; want %#v", state.Attack, wantAttack)
	}
	if state.PriorityHolder != "player-one" || state.PassCount != 0 || !state.PrioritySequenceOpen {
		t.Fatalf(
			"priority = holder %q, passes %d, open %t; want player-one, 0, true",
			state.PriorityHolder,
			state.PassCount,
			state.PrioritySequenceOpen,
		)
	}
	if state.Revision != 10 {
		t.Fatalf("Revision = %d; want 10", state.Revision)
	}
}

func TestDeclareAttackRecordsServantTarget(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p2-reversed"] = battleCard("p2-reversed", "player-two", model.OrientationReversed)
	state.Players[1].ServantZone = append(state.Players[1].ServantZone, "p2-reversed")

	err := DeclareAttack(
		&state,
		nil,
		"player-one",
		"p1-attacker",
		model.AttackTargetServant,
		"p2-defender",
		state.Revision,
	)
	if err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	if state.Attack.TargetKind != model.AttackTargetServant || state.Attack.TargetCardID != "p2-defender" {
		t.Fatalf("Attack = %#v; want servant target p2-defender", state.Attack)
	}
}

func TestDeclareAttackAllowsPlayerTargetWithPrintedHubrisDespiteReversed(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p2-reversed"] = battleCard("p2-reversed", "player-two", model.OrientationReversed)
	state.Players[1].ServantZone = append(state.Players[1].ServantZone, "p2-reversed")
	catalog := casterAetherCatalogForTest{
		"definition-p1-attacker": {
			ID:      "definition-p1-attacker",
			Type:    "Servant",
			Ability: "• [Hubris] (Cards with [Hubris] may attack enemy players even if there are reversed enemy servants.)",
		},
	}

	err := DeclareAttack(
		&state,
		catalog,
		"player-one",
		"p1-attacker",
		model.AttackTargetPlayer,
		"",
		state.Revision,
	)
	if err != nil {
		t.Fatalf("DeclareAttack() with Hubris error = %v", err)
	}
	if state.Attack.TargetKind != model.AttackTargetPlayer {
		t.Fatalf("Attack = %#v; want player target", state.Attack)
	}
}

func TestDeclareAttackRejectsInvalidRequestWithoutMutation(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(*model.MatchState)
		attackerID   model.MatchCardID
		targetKind   model.AttackTargetKind
		targetCardID model.MatchCardID
		wantErr      string
	}{
		{
			name:       "wrong phase",
			mutate:     func(state *model.MatchState) { state.Turn.Phase = model.PhaseMain },
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "must be in battle phase",
		},
		{
			name:       "non-active player",
			mutate:     func(state *model.MatchState) { state.PriorityHolder = "player-two" },
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "must be active player",
		},
		{
			name: "attack already in progress",
			mutate: func(state *model.MatchState) {
				state.Attack = model.AttackState{
					AttackerID: "p1-other",
					Step:       model.BattleStepDeclared,
				}
			},
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "already in progress",
		},
		{
			name: "closed priority",
			mutate: func(state *model.MatchState) {
				state.PrioritySequenceOpen = false
				state.PriorityHolder = ""
			},
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "priority sequence must be open",
		},
		{
			name: "rested attacker",
			mutate: func(state *model.MatchState) {
				state.CardInstances["p1-attacker"] = battleCard("p1-attacker", "player-one", model.OrientationRested)
			},
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "must be Recovered",
		},
		{
			name:         "player attack with card id",
			attackerID:   "p1-attacker",
			targetKind:   model.AttackTargetPlayer,
			targetCardID: "p2-defender",
			wantErr:      "player attacks cannot name a target card",
		},
		{
			name: "player attack blocked by reversed servant",
			mutate: func(state *model.MatchState) {
				state.CardInstances["p2-reversed"] = battleCard("p2-reversed", "player-two", model.OrientationReversed)
				state.Players[1].ServantZone = append(state.Players[1].ServantZone, "p2-reversed")
			},
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetPlayer,
			wantErr:    "reversed enemy servant",
		},
		{
			name:       "servant attack missing target",
			attackerID: "p1-attacker",
			targetKind: model.AttackTargetServant,
			wantErr:    "servant attacks require a target card",
		},
		{
			name:         "servant attack on non-enemy",
			attackerID:   "p1-attacker",
			targetKind:   model.AttackTargetServant,
			targetCardID: "p1-other",
			wantErr:      "not in the opponent's servant zone",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := battleStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneBattleState(state)
			actingPlayerID := model.PlayerID("player-one")
			if testCase.name == "non-active player" {
				actingPlayerID = "player-two"
			}

			err := DeclareAttack(
				&state,
				nil,
				actingPlayerID,
				testCase.attackerID,
				testCase.targetKind,
				testCase.targetCardID,
				state.Revision,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("DeclareAttack() error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected DeclareAttack mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestPassPriorityResolvesServantJudgmentAndDestroysWeakerTarget(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()

	if err := DeclareAttack(
		&state,
		nil,
		"player-one",
		"p1-attacker",
		model.AttackTargetServant,
		"p2-defender",
		state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	if err := PassPriority(&state, catalog, "player-one", state.Revision); err != nil {
		t.Fatalf("first pass error = %v", err)
	}
	if err := PassPriority(&state, catalog, "player-two", state.Revision); err != nil {
		t.Fatalf("second pass error = %v", err)
	}

	if state.Attack != (model.AttackState{}) {
		t.Fatalf("Attack = %#v; want cleared after judgment", state.Attack)
	}
	if !reflect.DeepEqual(state.Players[1].Graveyard, []model.MatchCardID{"p2-defender"}) {
		t.Fatalf("opponent Graveyard = %#v; want destroyed defender", state.Players[1].Graveyard)
	}
	if slicesContains(state.Players[1].ServantZone, "p2-defender") {
		t.Fatal("destroyed defender remained in ServantZone")
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != "player-one" || state.PassCount != 0 {
		t.Fatalf("post-judgment priority = open %t holder %q passes %d", state.PrioritySequenceOpen, state.PriorityHolder, state.PassCount)
	}
}

func TestPassPriorityLeavesEqualServantAlive(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	catalog["definition-p2-defender"] = gamecards.Card{
		ID: "definition-p2-defender", Type: "Servant", Attack: "3000", Defense: "1000",
	}

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetServant, "p2-defender", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if !slicesContains(state.Players[1].ServantZone, "p2-defender") {
		t.Fatal("equal-ATK defender was destroyed")
	}
	if len(state.Players[1].Graveyard) != 0 {
		t.Fatalf("Graveyard = %#v; want empty", state.Players[1].Graveyard)
	}
}

func TestPassPriorityUsesDefenseAgainstReversedServant(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.CardInstances["p2-defender"] = battleCard("p2-defender", "player-two", model.OrientationReversed)
	catalog["definition-p2-defender"] = gamecards.Card{
		ID: "definition-p2-defender", Type: "Servant", Attack: "1000", Defense: "4000",
	}

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetServant, "p2-defender", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if !slicesContains(state.Players[1].ServantZone, "p2-defender") {
		t.Fatal("reversed defender with higher DEF was destroyed")
	}
}

func TestPassPriorityAwaitsOrbChoiceOnPlayerAttack(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if state.Attack.Step != model.BattleStepAwaitingJudgment {
		t.Fatalf("Attack.Step = %q; want Awaiting Judgment", state.Attack.Step)
	}
	if state.Attack.CorruptCount != 1 {
		t.Fatalf("CorruptCount = %d; want 1 without Double Corrupt", state.Attack.CorruptCount)
	}
	if state.PrioritySequenceOpen || state.PriorityHolder != "" || state.PassCount != 0 {
		t.Fatal("orb-choice pause left priority open")
	}
	if !reflect.DeepEqual(state.Players[1].Orbs, []model.MatchCardID{"p2-orb-1", "p2-orb-2"}) {
		t.Fatalf("Orbs = %#v; want unchanged until CorruptOrb", state.Players[1].Orbs)
	}
}

func TestCorruptOrbChoosesSelectedEnemyOrb(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if err := CorruptOrb(&state, catalog, "player-one", 1, state.Revision); err != nil {
		t.Fatalf("CorruptOrb() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[1].Orbs, []model.MatchCardID{"p2-orb-1"}) {
		t.Fatalf("Orbs = %#v; want p2-orb-1 remaining after choosing index 1", state.Players[1].Orbs)
	}
	if !slicesContains(state.Players[1].Hand, "p2-orb-2") {
		t.Fatalf("Hand = %#v; want corrupted p2-orb-2", state.Players[1].Hand)
	}
	if state.Attack != (model.AttackState{}) {
		t.Fatalf("Attack = %#v; want cleared after corruption", state.Attack)
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != "player-one" || state.PassCount != 0 {
		t.Fatalf("post-corruption priority = open %t holder %q passes %d",
			state.PrioritySequenceOpen, state.PriorityHolder, state.PassCount)
	}
	if state.MatchStatus != model.StatusInProgress {
		t.Fatalf("MatchStatus = %q; corrupting an Orb must not finish the match", state.MatchStatus)
	}
}

func TestCorruptOrbRejectsInvalidChoiceWithoutMutation(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil
	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)
	beforeRevision := state.Revision
	beforeOrbs := append([]model.MatchCardID(nil), state.Players[1].Orbs...)

	err := CorruptOrb(&state, catalog, "player-one", 2, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("CorruptOrb() error = %v; want out-of-range", err)
	}
	if state.Revision != beforeRevision || !reflect.DeepEqual(state.Players[1].Orbs, beforeOrbs) {
		t.Fatal("rejected CorruptOrb mutated state")
	}
}

func TestPassPriorityWinsWhenPlayerAttackFindsZeroOrbs(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()
	state.Players[1].ServantZone = nil
	state.Players[1].Orbs = nil

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	passTwiceForTest(t, &state, catalog)

	if state.MatchStatus != model.StatusFinished {
		t.Fatalf("MatchStatus = %q; want Finished", state.MatchStatus)
	}
	want := model.MatchResult{
		Winner: "player-one",
		Loser:  "player-two",
		Reason: model.EndReasonZeroOrbs,
	}
	if state.Result != want {
		t.Fatalf("Result = %#v; want %#v", state.Result, want)
	}
	if state.PrioritySequenceOpen || state.PriorityHolder != "" {
		t.Fatal("finished match reopened priority")
	}
}

func TestPassPriorityClearsInterruptedAttackWithoutJudgment(t *testing.T) {
	state, catalog := battleJudgmentStateForTest()

	if err := DeclareAttack(&state, nil, "player-one", "p1-attacker", model.AttackTargetServant, "p2-defender", state.Revision,
	); err != nil {
		t.Fatalf("DeclareAttack() error = %v", err)
	}
	attacker := state.CardInstances["p1-attacker"]
	attacker.Orientation = model.OrientationReversed
	state.CardInstances["p1-attacker"] = attacker

	passTwiceForTest(t, &state, catalog)

	if state.Attack != (model.AttackState{}) {
		t.Fatalf("Attack = %#v; want cleared after interruption", state.Attack)
	}
	if slicesContains(state.Players[1].Graveyard, "p2-defender") {
		t.Fatal("interrupted attack still destroyed the defender")
	}
}

func battleStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-attacker": battleCard("p1-attacker", "player-one", model.OrientationRecovered),
			"p1-other":    battleCard("p1-other", "player-one", model.OrientationRecovered),
			"p2-defender": battleCard("p2-defender", "player-two", model.OrientationRecovered),
			"p2-orb-1":    battleCard("p2-orb-1", "player-two", model.OrientationRecovered),
			"p2-orb-2":    battleCard("p2-orb-2", "player-two", model.OrientationRecovered),
		},
		Players: [2]model.PlayerState{
			{
				ID:          "player-one",
				ServantZone: []model.MatchCardID{"p1-attacker", "p1-other"},
			},
			{
				ID:          "player-two",
				ServantZone: []model.MatchCardID{"p2-defender"},
				Orbs:        []model.MatchCardID{"p2-orb-1", "p2-orb-2"},
			},
		},
		FirstPlayer:          "player-one",
		MatchStatus:          model.StatusInProgress,
		Revision:             9,
		Turn:                 model.TurnState{Number: 2, ActivePlayer: "player-one", Phase: model.PhaseBattle},
		PriorityHolder:       "player-one",
		PrioritySequenceOpen: true,
	}
}

func battleJudgmentStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := battleStateForTest()
	catalog := casterAetherCatalogForTest{
		"definition-p1-attacker": {
			ID: "definition-p1-attacker", Type: "Servant", Attack: "3000", Defense: "2000",
		},
		"definition-p2-defender": {
			ID: "definition-p2-defender", Type: "Servant", Attack: "2000", Defense: "1000",
		},
	}
	return state, catalog
}

func passTwiceForTest(t *testing.T, state *model.MatchState, catalog casterAetherCatalogForTest) {
	t.Helper()
	if err := PassPriority(state, catalog, state.PriorityHolder, state.Revision); err != nil {
		t.Fatalf("first pass error = %v", err)
	}
	if err := PassPriority(state, catalog, state.PriorityHolder, state.Revision); err != nil {
		t.Fatalf("second pass error = %v", err)
	}
}

func slicesContains(ids []model.MatchCardID, want model.MatchCardID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func battleCard(
	matchID model.MatchCardID,
	controller model.PlayerID,
	orientation model.CardOrientation,
) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CardID("definition-" + string(matchID)),
		MatchID:      matchID,
		Owner:        controller,
		Controller:   controller,
		CardCategory: model.CategoryPrintedCard,
		Face:         model.CardFaceUp,
		Orientation:  orientation,
	}
}

func cloneBattleState(state model.MatchState) model.MatchState {
	clone := state
	clone.ChaseLinks = append(model.Chase(nil), state.ChaseLinks...)
	clone.CardInstances = make(map[model.MatchCardID]model.CardInstance, len(state.CardInstances))
	for id, instance := range state.CardInstances {
		clone.CardInstances[id] = instance
	}
	for index := range state.Players {
		clone.Players[index].ServantZone = append([]model.MatchCardID(nil), state.Players[index].ServantZone...)
		clone.Players[index].Orbs = append([]model.MatchCardID(nil), state.Players[index].Orbs...)
	}
	return clone
}

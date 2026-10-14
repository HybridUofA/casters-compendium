package engine

import (
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func clearTurnEventFlags(state *model.MatchState) {
	if state == nil {
		return
	}
	state.TurnFlags = model.TurnFlags{}
	state.ContinuousUntilEOT = model.ContinuousUntilEOT{}
	for id, instance := range state.CardInstances {
		if !instance.BattledThisTurn && !instance.GrantedSlowStartUntilEOT && !instance.LosesSlowStartUntilEOT {
			continue
		}
		instance.BattledThisTurn = false
		instance.GrantedSlowStartUntilEOT = false
		instance.LosesSlowStartUntilEOT = false
		state.CardInstances[id] = instance
	}
}

func playerFlagIndex(state *model.MatchState, playerID model.PlayerID) int {
	if state == nil {
		return -1
	}
	for index, player := range state.Players {
		if player.ID == playerID {
			return index
		}
	}
	return -1
}

func noteServantEntered(state *model.MatchState, controller model.PlayerID, cardID model.MatchCardID) {
	if state == nil {
		return
	}
	if index := playerFlagIndex(state, controller); index >= 0 {
		state.TurnFlags.ServantsEntered[index]++
	}
	if instance, ok := state.CardInstances[cardID]; ok {
		instance.EnteredOnTurn = state.Turn.Number
		state.CardInstances[cardID] = instance
	}
}

func noteCardPlayed(state *model.MatchState, catalog rules.CardCatalog, controller model.PlayerID, printedID model.CardID) {
	if state == nil {
		return
	}
	index := playerFlagIndex(state, controller)
	if index < 0 {
		return
	}
	state.TurnFlags.AnyCardPlayed[index]++
	if catalog == nil {
		return
	}
	definition, found := catalog.FindByID(string(printedID))
	if !found {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(definition.Type), "caster") {
		state.TurnFlags.NonCasterPlayed[index]++
	}
}

func noteBreakPlayed(state *model.MatchState, controller model.PlayerID) {
	if state == nil {
		return
	}
	if index := playerFlagIndex(state, controller); index >= 0 {
		state.TurnFlags.BreakPlayed[index]++
	}
}

func noteAttackDeclared(state *model.MatchState, attackerController model.PlayerID, attackerID, targetCardID model.MatchCardID) {
	if state == nil {
		return
	}
	if instance, ok := state.CardInstances[attackerID]; ok {
		instance.BattledThisTurn = true
		state.CardInstances[attackerID] = instance
	}
	if targetCardID != "" {
		if instance, ok := state.CardInstances[targetCardID]; ok {
			instance.BattledThisTurn = true
			state.CardInstances[targetCardID] = instance
		}
	}
	attackerIndex := playerFlagIndex(state, attackerController)
	if attackerIndex < 0 {
		return
	}
	enemyIndex := 1 - attackerIndex
	state.TurnFlags.EnemyAttacks[enemyIndex]++
}

func noteCorrupt(state *model.MatchState, actingPlayerID model.PlayerID) {
	if state == nil {
		return
	}
	if index := playerFlagIndex(state, actingPlayerID); index >= 0 {
		state.TurnFlags.AlliedCorruptedOrb[index]++
	}
}

func noteConjureDraw(state *model.MatchState, controller model.PlayerID, count int) {
	if state == nil || count <= 0 {
		return
	}
	if index := playerFlagIndex(state, controller); index >= 0 {
		state.TurnFlags.ConjureEffectDraws[index] += count
	}
}

func noteEnterAbility(state *model.MatchState, controller model.PlayerID) {
	if state == nil {
		return
	}
	controllerIndex := playerFlagIndex(state, controller)
	if controllerIndex < 0 {
		return
	}
	enemyIndex := 1 - controllerIndex
	state.TurnFlags.EnemyEnterAbility[enemyIndex]++
}

func noteDiscard(state *model.MatchState, discardingPlayerID model.PlayerID) {
	if state == nil {
		return
	}
	discarderIndex := playerFlagIndex(state, discardingPlayerID)
	if discarderIndex < 0 {
		return
	}
	enemyIndex := 1 - discarderIndex
	state.TurnFlags.EnemyDiscarded[enemyIndex]++
}

func noteReturnToHand(state *model.MatchState, returnedCardID model.MatchCardID) {
	if state == nil {
		return
	}
	instance, ok := state.CardInstances[returnedCardID]
	if !ok {
		return
	}
	controllerIndex := playerFlagIndex(state, instance.Controller)
	if controllerIndex < 0 {
		return
	}
	// From the opponent's perspective an "enemy servant" returned.
	state.TurnFlags.EnemyReturnedToHand[1-controllerIndex]++
}

type destroyCause string

const (
	destroyCauseBattle destroyCause = "battle"
	destroyCauseEffect destroyCause = "effect"
)

func noteDestroyed(
	state *model.MatchState,
	catalog rules.CardCatalog,
	destroyed model.CardInstance,
	cause destroyCause,
	destroyerController model.PlayerID,
) {
	if state == nil {
		return
	}
	ownerIndex := playerFlagIndex(state, destroyed.Controller)
	if ownerIndex >= 0 {
		switch cause {
		case destroyCauseBattle:
			state.TurnFlags.AlliedDestroyedInBattle[ownerIndex]++
		default:
			state.TurnFlags.AlliedDestroyedNonBattle[ownerIndex]++
		}
		if cardHasTrait(catalog, destroyed.CardID, "insect") {
			state.TurnFlags.AlliedInsectDestroyedOrBanished[ownerIndex]++
		}
	}
	if cause == destroyCauseBattle {
		if index := playerFlagIndex(state, destroyerController); index >= 0 {
			state.TurnFlags.BattleDestroysDealt[index]++
		}
	} else if destroyerController != "" {
		if index := playerFlagIndex(state, destroyerController); index >= 0 {
			state.TurnFlags.SpellDestroyedEnemy[index]++
		}
	}
}

func noteBanished(state *model.MatchState, catalog rules.CardCatalog, cardID model.MatchCardID) {
	if state == nil {
		return
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return
	}
	index := playerFlagIndex(state, instance.Controller)
	if index < 0 {
		return
	}
	if cardHasTrait(catalog, instance.CardID, "insect") {
		state.TurnFlags.AlliedInsectDestroyedOrBanished[index]++
	}
}

func cardHasTrait(catalog rules.CardCatalog, printedID model.CardID, trait string) bool {
	if catalog == nil {
		return false
	}
	definition, found := catalog.FindByID(string(printedID))
	if !found {
		return false
	}
	return traitsContain(definition, trait)
}

func traitsContain(definition gamecards.Card, trait string) bool {
	needle := strings.ToLower(strings.TrimSpace(trait))
	if needle == "" {
		return false
	}
	for _, part := range strings.FieldsFunc(definition.Traits, func(r rune) bool {
		return r == ',' || r == '/' || r == ';'
	}) {
		if strings.EqualFold(strings.TrimSpace(part), needle) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(definition.Traits), needle)
}

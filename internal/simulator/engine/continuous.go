package engine

import (
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// servantCannotBeAttacked reports continuous / until-EOT protection on a servant.
func servantCannotBeAttacked(
	state *model.MatchState,
	catalog rules.CardCatalog,
	targetID model.MatchCardID,
	targetControllerIndex int,
) bool {
	if state == nil {
		return false
	}
	instance, ok := state.CardInstances[targetID]
	if !ok {
		return false
	}
	if state.ContinuousUntilEOT.AlliedServantsCannotBeAttacked[targetControllerIndex] {
		return true
	}
	if state.ContinuousUntilEOT.RestedAlliedCannotBeAttacked[targetControllerIndex] &&
		countServants(state, catalog, targetControllerIndex) < countServants(state, catalog, 1-targetControllerIndex) {
		return true
	}
	if instance.Orientation == model.OrientationRested {
		if fieldGrantsRestedAlliedProtection(state, catalog, targetControllerIndex) {
			return true
		}
		if selfCannotBeAttackedWhileRested(state, catalog, targetID) {
			return true
		}
	}
	if selfCannotBeAttackedByServantsWhileRested(state, catalog, targetID) &&
		instance.Orientation == model.OrientationRested {
		return true
	}
	return false
}

func playerCannotBeAttacked(state *model.MatchState, opponentIndex int) bool {
	if state == nil {
		return false
	}
	return state.ContinuousUntilEOT.PlayerCannotBeAttacked[opponentIndex]
}

func fieldGrantsRestedAlliedProtection(
	state *model.MatchState,
	catalog rules.CardCatalog,
	controllerIndex int,
) bool {
	if catalog == nil {
		return false
	}
	for _, cardID := range state.Players[controllerIndex].ServantZone {
		instance, ok := state.CardInstances[cardID]
		if !ok {
			continue
		}
		definition, found := catalog.FindByID(string(instance.CardID))
		if !found {
			continue
		}
		normalized := strings.ToLower(definition.Ability)
		// Barrier of Arrogance-style: continuous while the barrier is in play.
		if strings.Contains(normalized, "rested allied servants can't be attacked") ||
			strings.Contains(normalized, "rested allied servants cannot be attacked") {
			return true
		}
		// Pompous Delphinium-style: only while this card is Reversed.
		if strings.Contains(normalized, "while i am reversed") &&
			(strings.Contains(normalized, "cannot attack allied rested") ||
				strings.Contains(normalized, "can't attack allied rested")) &&
			instance.Orientation == model.OrientationReversed {
			return true
		}
	}
	return false
}

func selfCannotBeAttackedWhileRested(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
) bool {
	ability := printedAbility(state, catalog, cardID)
	normalized := strings.ToLower(ability)
	return strings.Contains(normalized, "i cannot be attacked while rested") ||
		strings.Contains(normalized, "cannot be attacked while rested")
}

func selfCannotBeAttackedByServantsWhileRested(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
) bool {
	ability := printedAbility(state, catalog, cardID)
	normalized := strings.ToLower(ability)
	return strings.Contains(normalized, "cannot be attacked by servants while rested")
}

func printedAbility(state *model.MatchState, catalog rules.CardCatalog, cardID model.MatchCardID) string {
	if state == nil || catalog == nil {
		return ""
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return ""
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return ""
	}
	return definition.Ability
}

// hasSlowStartThisTurn is true when the attacker cannot attack due to Slow Start.
func hasSlowStartThisTurn(
	state *model.MatchState,
	catalog rules.CardCatalog,
	attackerID model.MatchCardID,
) bool {
	if state == nil {
		return false
	}
	instance, ok := state.CardInstances[attackerID]
	if !ok {
		return false
	}
	if instance.LosesSlowStartUntilEOT {
		return false
	}
	if instance.GrantedSlowStartUntilEOT {
		return true
	}
	if fieldGrantsGlobalSlowStart(state, catalog) {
		if instance.EnteredOnTurn == state.Turn.Number {
			return true
		}
	}
	ability := printedAbility(state, catalog, attackerID)
	if abilityHasPrintedSlowStart(ability) && instance.EnteredOnTurn == state.Turn.Number {
		return true
	}
	return false
}

func abilityHasPrintedSlowStart(ability string) bool {
	normalized := strings.ToLower(ability)
	return strings.Contains(normalized, "[slow start]") ||
		strings.Contains(normalized, "slow start")
}

func fieldGrantsGlobalSlowStart(state *model.MatchState, catalog rules.CardCatalog) bool {
	if state == nil || catalog == nil {
		return false
	}
	for _, player := range state.Players {
		for _, cardID := range player.ServantZone {
			instance, ok := state.CardInstances[cardID]
			if !ok {
				continue
			}
			definition, found := catalog.FindByID(string(instance.CardID))
			if !found {
				continue
			}
			normalized := strings.ToLower(definition.Ability)
			// Sophomore Slump-style continuous barrier.
			if strings.Contains(normalized, "all servants gain slow start") {
				return true
			}
		}
	}
	return false
}

// worldOfRageAffects reports whether controllerIndex's servants are under an
// enemy World of Rage (enter Recovered; cannot Switch except by effect).
func worldOfRageAffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	controllerIndex int,
) bool {
	if state == nil || catalog == nil || controllerIndex < 0 || controllerIndex > 1 {
		return false
	}
	enemyIndex := 1 - controllerIndex
	for _, cardID := range state.Players[enemyIndex].ServantZone {
		instance, ok := state.CardInstances[cardID]
		if !ok {
			continue
		}
		definition, found := catalog.FindByID(string(instance.CardID))
		if !found {
			continue
		}
		normalized := strings.ToLower(definition.Ability)
		if strings.Contains(normalized, "enemy servants enter the field recovered") ||
			strings.Contains(normalized, "enemy servants cannot be switched") {
			return true
		}
	}
	return false
}

// cardsLoseBreak reports Saaya-style continuous: while an opponent controls a
// card that says opponents with 6+ orbs lose Break, and this owner has 6+ orbs.
func cardsLoseBreak(
	state *model.MatchState,
	catalog rules.CardCatalog,
	ownerIndex int,
) bool {
	if state == nil || catalog == nil || ownerIndex < 0 || ownerIndex > 1 {
		return false
	}
	if len(state.Players[ownerIndex].Orbs) < 6 {
		return false
	}
	enemyIndex := 1 - ownerIndex
	for _, cardID := range state.Players[enemyIndex].CasterZone {
		if abilityMakesOpponentLoseBreak(state, catalog, cardID) {
			return true
		}
	}
	for _, cardID := range state.Players[enemyIndex].ServantZone {
		if abilityMakesOpponentLoseBreak(state, catalog, cardID) {
			return true
		}
	}
	return false
}

func abilityMakesOpponentLoseBreak(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
) bool {
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return false
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return false
	}
	normalized := strings.ToLower(definition.Ability)
	return strings.Contains(normalized, "lose break") ||
		strings.Contains(normalized, "loses break")
}

package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// ModifyAttack adds delta to the servant's AttackModifier (printed ATK is unchanged).
func ModifyAttack(state *model.MatchState, cardID model.MatchCardID, delta int) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("card %q is not in a servant zone", cardID)
	}
	instance.AttackModifier += delta
	state.CardInstances[cardID] = instance
	return nil
}

// ModifyDefense adds delta to the servant's DefenseModifier (printed DEF is unchanged).
func ModifyDefense(state *model.MatchState, cardID model.MatchCardID, delta int) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("card %q is not in a servant zone", cardID)
	}
	instance.DefenseModifier += delta
	state.CardInstances[cardID] = instance
	return nil
}

// SetCombatModifiers replaces AttackModifier and DefenseModifier on a Servant.
// Recipes compute absolute "become X/Y" or swap math from printed ATK/DEF.
func SetCombatModifiers(state *model.MatchState, cardID model.MatchCardID, attackMod, defenseMod int) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("card %q is not in a servant zone", cardID)
	}
	instance.AttackModifier = attackMod
	instance.DefenseModifier = defenseMod
	state.CardInstances[cardID] = instance
	return nil
}

// SwapCombatModifiers exchanges AttackModifier and DefenseModifier.
// For printed ATK/DEF swaps (Lumielle), the recipe should call SetCombatModifiers
// with values derived from printed stats + current modifiers.
func SwapCombatModifiers(state *model.MatchState, cardID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("card %q is not in a servant zone", cardID)
	}
	instance.AttackModifier, instance.DefenseModifier = instance.DefenseModifier, instance.AttackModifier
	state.CardInstances[cardID] = instance
	return nil
}

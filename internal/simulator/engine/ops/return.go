package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// ReturnToHand moves a Servant or Caster from the field to its owner's hand.
func ReturnToHand(state *model.MatchState, cardID model.MatchCardID) error {
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
	if location.Zone != model.ZoneServant && location.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not in a field zone that can return to hand", cardID)
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(&instance)
	state.Players[ownerIndex].Hand = append(state.Players[ownerIndex].Hand, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

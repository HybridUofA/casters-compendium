package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Exile moves cardID from its current zone to its owner's Exile zone.
func Exile(state *model.MatchState, cardID model.MatchCardID) error {
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
	if location.Zone == model.ZoneExile {
		return fmt.Errorf("card %q is already exiled", cardID)
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(&instance)
	state.Players[ownerIndex].Exile = append(state.Players[ownerIndex].Exile, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

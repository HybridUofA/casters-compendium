package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func recoverPlayerCards(state *model.MatchState, playerID model.PlayerID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	activeIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == playerID {
			activeIndex = index
		}
	}
	if activeIndex == -1 {
		return fmt.Errorf("player %q not found in match state", playerID)
	}
	for _, cardID := range state.Players[activeIndex].CasterZone {
		instance, exists := state.CardInstances[cardID]
		if !exists {
			return fmt.Errorf("card %q does not exist", cardID)
		}
		if instance.Controller != playerID {
			return fmt.Errorf("card %q does not belong to %q", cardID, playerID)
		}
	}
	for _, cardID := range state.Players[activeIndex].ServantZone {
		instance, exists := state.CardInstances[cardID]
		if !exists {
			return fmt.Errorf("card %q does not exist", cardID)
		}
		if instance.Controller != playerID {
			return fmt.Errorf("card %q does not belong to %q", cardID, playerID)
		}
	}
	for _, cardID := range state.Players[activeIndex].CasterZone {
		instance, exists := state.CardInstances[cardID]
		if !exists {
			return fmt.Errorf("card %q does not exist", cardID)
		}
		if instance.Orientation == model.OrientationRested {
			instance.Orientation = model.OrientationRecovered
			state.CardInstances[cardID] = instance
		}
	}
	for _, cardID := range state.Players[activeIndex].ServantZone {
		instance, exists := state.CardInstances[cardID]
		if !exists {
			return fmt.Errorf("card %q does not exist", cardID)
		}
		if instance.Orientation == model.OrientationRested {
			instance.Orientation = model.OrientationRecovered
			state.CardInstances[cardID] = instance
		}
	}
	return nil
}

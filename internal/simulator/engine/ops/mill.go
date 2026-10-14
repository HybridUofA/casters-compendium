package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Mill moves the top count cards of playerID's deck into their graveyard.
func Mill(state *model.MatchState, playerID model.PlayerID, count int) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	taken, remaining, err := takeTopCards(state.Players[index].Deck, count)
	if err != nil {
		return fmt.Errorf("mill: %w", err)
	}
	state.Players[index].Deck = remaining
	for _, cardID := range taken {
		if instance, ok := state.CardInstances[cardID]; ok {
			clearFieldStatus(&instance)
			state.CardInstances[cardID] = instance
		}
	}
	state.Players[index].Graveyard = append(state.Players[index].Graveyard, taken...)
	return nil
}

// Discard moves a hand card to its owner's graveyard.
func Discard(state *model.MatchState, cardID model.MatchCardID) error {
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
	if location.Zone != model.ZoneHand {
		return fmt.Errorf("card %q is not in hand", cardID)
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(&instance)
	state.Players[ownerIndex].Graveyard = append(state.Players[ownerIndex].Graveyard, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

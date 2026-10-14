package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Draw moves the top count cards of playerID's deck into their hand.
func Draw(state *model.MatchState, playerID model.PlayerID, count int) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	taken, remaining, err := takeTopCards(state.Players[index].Deck, count)
	if err != nil {
		return fmt.Errorf("error taking cards: %w", err)
	}
	state.Players[index].Deck = remaining
	state.Players[index].Hand = append(state.Players[index].Hand, taken...)
	return nil
}

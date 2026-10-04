package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func shuffleDeck(deck []model.MatchCardID, random RandomSource) error {
	for index := len(deck) - 1; index > 0; index-- {
		swapIndex := random.RandInt(index + 1)
		if swapIndex < 0 || swapIndex > index {
			return fmt.Errorf("%d is an invalid index value. Must be between 0 and %d", swapIndex, index)
		}
		deck[index], deck[swapIndex] = deck[swapIndex], deck[index]
	}
	return nil
}

// ShufflePlayerDeck shuffles the acting player's deck in place. Used after
// Cockatrice-style deck browse so looked-at order is not retained.
func ShufflePlayerDeck(
	state *model.MatchState,
	random RandomSource,
	actingPlayerID model.PlayerID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if random == nil {
		return fmt.Errorf("random source cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	playerIndex := -1
	for index, player := range state.Players {
		if player.ID == actingPlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("player not found")
	}
	if err := shuffleDeck(state.Players[playerIndex].Deck, random); err != nil {
		return fmt.Errorf("shuffle deck: %w", err)
	}
	state.Revision++
	return nil
}

package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func DrawCards(state *model.MatchState, playerID model.PlayerID, count int) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if strings.TrimSpace(string(playerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	playerIndex := -1
	for index, player := range state.Players {
		if player.ID == playerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("player not found")
	}
	taken, remaining, err := takeTopCards(state.Players[playerIndex].Deck, count)
	if err != nil {
		return fmt.Errorf("error taking cards: %w", err)
	}
	state.Players[playerIndex].Deck = remaining
	state.Players[playerIndex].Hand = append(state.Players[playerIndex].Hand, taken...)
	return nil
}

// ManualDrawCards is the Cockatrice-style player command: draw N from your deck
// outside the Draw Phase automation. It validates revision and bumps it.
// When the drawer controls Sage Advice, the draw pauses for an optional dig.
func ManualDrawCards(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	count int,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if count < 1 {
		return fmt.Errorf("draw count must be at least 1")
	}
	if state.PendingDraw.Step != model.PendingDrawIdle {
		return fmt.Errorf("resolve the pending draw before drawing again")
	}
	if err := beginDrawOrOfferReplacement(state, catalog, actingPlayerID, count); err != nil {
		return err
	}
	state.Revision++
	return nil
}

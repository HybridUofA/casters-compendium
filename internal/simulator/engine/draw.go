package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// DrawCards moves the top count cards of playerID's deck into their hand.
func DrawCards(state *model.MatchState, playerID model.PlayerID, count int) error {
	return ops.Draw(state, playerID, count)
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

package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// ManualCorruptOrb moves an Orb into its owner's hand for effect resolution
// (Straw Effigy, Vega Scarlet Star, etc.). Unlike battle CorruptOrbs, this does
// not require a player-attack judgment. Break is offered when the card prints it.
func ManualCorruptOrb(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	orbID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if strings.TrimSpace(string(orbID)) == "" {
		return fmt.Errorf("orb ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	location, err := model.FindCardLocation(state, orbID)
	if err != nil {
		return fmt.Errorf("locate orb: %w", err)
	}
	if location.Zone != model.ZoneOrbs {
		return fmt.Errorf("card %q is not an Orb", orbID)
	}
	ownerID := state.Players[location.PlayerIndex].ID
	if err := ops.MoveOrbToHand(state, ownerID, location.CardIndex); err != nil {
		return err
	}
	model.MarkCardKnown(state, ownerID, orbID)
	noteCorrupt(state, actingPlayerID)
	queueBreakIfPresent(state, catalog, ownerID, orbID)
	state.Revision++
	return nil
}

// ManualBanishOrb banishes an Orb to its owner's graveyard (Vega cost, etc.).
func ManualBanishOrb(
	state *model.MatchState,
	_ rules.CardCatalog,
	actingPlayerID model.PlayerID,
	orbID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if strings.TrimSpace(string(orbID)) == "" {
		return fmt.Errorf("orb ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	location, err := model.FindCardLocation(state, orbID)
	if err != nil {
		return fmt.Errorf("locate orb: %w", err)
	}
	if location.Zone != model.ZoneOrbs {
		return fmt.Errorf("card %q is not an Orb", orbID)
	}
	if err := ops.Banish(state, orbID); err != nil {
		return err
	}
	state.Revision++
	return nil
}

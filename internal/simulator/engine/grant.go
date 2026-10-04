package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// SetGrantedDoubleCorrupt toggles a manual Double Corrupt marker on a Servant.
// Use this for caster abilities and other temporary grants that are not printed
// on the Servant's ability text (Hilde Willow-style resolution).
func SetGrantedDoubleCorrupt(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	enabled bool,
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
	if strings.TrimSpace(string(cardID)) == "" {
		return fmt.Errorf("card ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
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
		return fmt.Errorf("double corrupt can only be granted to a servant on the field")
	}
	if !slices.Contains(state.Players[location.PlayerIndex].ServantZone, cardID) {
		return fmt.Errorf("card %q is not in a servant zone", cardID)
	}
	if instance.GrantedDoubleCorrupt == enabled {
		return fmt.Errorf("granted double corrupt is already %t", enabled)
	}
	instance.GrantedDoubleCorrupt = enabled
	state.CardInstances[cardID] = instance
	state.Revision++
	return nil
}

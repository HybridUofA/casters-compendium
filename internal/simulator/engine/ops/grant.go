package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// GrantDoubleCorrupt sets or clears the Double Corrupt marker on a Servant.
func GrantDoubleCorrupt(state *model.MatchState, cardID model.MatchCardID, enabled bool) error {
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
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("double corrupt can only be granted to a servant on the field")
	}
	instance.GrantedDoubleCorrupt = enabled
	state.CardInstances[cardID] = instance
	return nil
}

package ops

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// AttachStock removes stockCardID from its current zone and appends it under hostID's Stock.
// hostID must be on the field (Caster or Servant).
func AttachStock(state *model.MatchState, hostID, stockCardID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if hostID == stockCardID {
		return fmt.Errorf("card cannot be stocked under itself")
	}
	host, ok := state.CardInstances[hostID]
	if !ok {
		return fmt.Errorf("host %q does not exist", hostID)
	}
	hostLoc, err := model.FindCardLocation(state, hostID)
	if err != nil {
		return fmt.Errorf("locate host: %w", err)
	}
	if hostLoc.Zone != model.ZoneServant && hostLoc.Zone != model.ZoneCaster {
		return fmt.Errorf("host %q is not on the field", hostID)
	}
	stock, ok := state.CardInstances[stockCardID]
	if !ok {
		return fmt.Errorf("stock card %q does not exist", stockCardID)
	}
	stockLoc, err := model.FindCardLocation(state, stockCardID)
	if err != nil {
		return fmt.Errorf("locate stock card: %w", err)
	}
	if err := removeCardAt(stockLoc, state); err != nil {
		return err
	}
	clearFieldStatus(&stock)
	host.Stock = append(slices.Clone(host.Stock), stockCardID)
	state.CardInstances[hostID] = host
	state.CardInstances[stockCardID] = stock
	return nil
}

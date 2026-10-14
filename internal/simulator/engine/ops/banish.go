package ops

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Banish moves cardID to its owner's graveyard (CR 10.8).
// Valid sources: Servant zone, Caster zone, Orbs, or Stock under a field card.
// Hand→GY is Discard; Remove From the Game is Exile.
func Banish(state *model.MatchState, cardID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, locErr := model.FindCardLocation(state, cardID)
	if locErr == nil {
		switch location.Zone {
		case model.ZoneServant, model.ZoneCaster, model.ZoneOrbs:
			if err := removeCardAt(location, state); err != nil {
				return err
			}
			return appendToOwnerGraveyard(state, cardID, &instance)
		default:
			return fmt.Errorf("card %q in zone %q cannot be banished (use Discard or Exile)", cardID, location.Zone)
		}
	}
	return BanishStock(state, cardID)
}

// BanishStock removes stockCardID from its host's Stock and puts it in the owner's graveyard.
func BanishStock(state *model.MatchState, stockCardID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[stockCardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", stockCardID)
	}
	hostID, stockIndex, err := findStockHost(state, stockCardID)
	if err != nil {
		return err
	}
	host := state.CardInstances[hostID]
	host.Stock = slices.Delete(slices.Clone(host.Stock), stockIndex, stockIndex+1)
	state.CardInstances[hostID] = host
	return appendToOwnerGraveyard(state, stockCardID, &instance)
}

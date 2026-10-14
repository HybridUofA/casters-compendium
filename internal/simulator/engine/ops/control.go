package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// GainControl moves a field Servant or Caster into newController's matching zone
// and sets Controller (CR 10.16). Owner is unchanged.
func GainControl(state *model.MatchState, cardID model.MatchCardID, newController model.PlayerID) error {
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
	if location.Zone != model.ZoneServant && location.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not in a controllable field zone", cardID)
	}
	destIndex, err := playerIndex(state, newController)
	if err != nil {
		return err
	}
	if state.Players[location.PlayerIndex].ID == newController {
		instance.Controller = newController
		state.CardInstances[cardID] = instance
		return nil
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	instance.Controller = newController
	destZone, err := zoneCards(state, destIndex, location.Zone)
	if err != nil {
		return err
	}
	*destZone = append(*destZone, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

// ExchangeControl swaps control of two field cards (CR 10.17).
// Both must be in Servant or Caster zones (zones may differ).
func ExchangeControl(state *model.MatchState, cardA, cardB model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if cardA == cardB {
		return fmt.Errorf("cannot exchange a card with itself")
	}
	instA, okA := state.CardInstances[cardA]
	instB, okB := state.CardInstances[cardB]
	if !okA {
		return fmt.Errorf("card %q does not exist", cardA)
	}
	if !okB {
		return fmt.Errorf("card %q does not exist", cardB)
	}
	locA, err := model.FindCardLocation(state, cardA)
	if err != nil {
		return fmt.Errorf("locate card A: %w", err)
	}
	locB, err := model.FindCardLocation(state, cardB)
	if err != nil {
		return fmt.Errorf("locate card B: %w", err)
	}
	if locA.Zone != model.ZoneServant && locA.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not in a controllable field zone", cardA)
	}
	if locB.Zone != model.ZoneServant && locB.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not in a controllable field zone", cardB)
	}
	ctrlA := instA.Controller
	ctrlB := instB.Controller
	if ctrlA == ctrlB {
		return fmt.Errorf("both cards are already controlled by %q", ctrlA)
	}

	// Remove both first so indices stay valid, then place into the other's controller zone.
	zoneA := locA.Zone
	zoneB := locB.Zone
	if err := removeCardAt(locA, state); err != nil {
		return err
	}
	// locB may have shifted if same player/zone and A was before B.
	locB, err = model.FindCardLocation(state, cardB)
	if err != nil {
		return fmt.Errorf("re-locate card B: %w", err)
	}
	if err := removeCardAt(locB, state); err != nil {
		return err
	}

	destA, err := playerIndex(state, ctrlB)
	if err != nil {
		return err
	}
	destB, err := playerIndex(state, ctrlA)
	if err != nil {
		return err
	}
	sliceA, err := zoneCards(state, destA, zoneA)
	if err != nil {
		return err
	}
	sliceB, err := zoneCards(state, destB, zoneB)
	if err != nil {
		return err
	}
	*sliceA = append(*sliceA, cardA)
	*sliceB = append(*sliceB, cardB)
	instA.Controller = ctrlB
	instB.Controller = ctrlA
	state.CardInstances[cardA] = instA
	state.CardInstances[cardB] = instB
	return nil
}

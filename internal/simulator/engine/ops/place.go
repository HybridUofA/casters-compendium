package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// MoveToHand moves cardID from its current public/hidden zone into its owner's hand.
// Stocked cards must be Banished or otherwise detached first.
func MoveToHand(state *model.MatchState, cardID model.MatchCardID) error {
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
	if location.Zone == model.ZoneHand {
		return fmt.Errorf("card %q is already in hand", cardID)
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(&instance)
	state.Players[ownerIndex].Hand = append(state.Players[ownerIndex].Hand, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

// PutAsServant moves cardID into controller's Servant zone.
func PutAsServant(
	state *model.MatchState,
	cardID model.MatchCardID,
	controller model.PlayerID,
	face model.CardFace,
	orientation model.CardOrientation,
) error {
	return placeOnField(state, cardID, controller, model.ZoneServant, face, orientation)
}

// PutAsCaster moves cardID into controller's Caster zone.
func PutAsCaster(
	state *model.MatchState,
	cardID model.MatchCardID,
	controller model.PlayerID,
	face model.CardFace,
	orientation model.CardOrientation,
) error {
	return placeOnField(state, cardID, controller, model.ZoneCaster, face, orientation)
}

// PutAsOrb moves cardID into playerID's Orb zone face down (CR-style "as an orb").
func PutAsOrb(state *model.MatchState, cardID model.MatchCardID, playerID model.PlayerID) error {
	return placeOnField(state, cardID, playerID, model.ZoneOrbs, model.CardFaceDown, model.OrientationRecovered)
}

func placeOnField(
	state *model.MatchState,
	cardID model.MatchCardID,
	controller model.PlayerID,
	zone model.Zone,
	face model.CardFace,
	orientation model.CardOrientation,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if zone != model.ZoneServant && zone != model.ZoneCaster && zone != model.ZoneOrbs {
		return fmt.Errorf("unsupported field zone %q", zone)
	}
	if face != model.CardFaceUp && face != model.CardFaceDown {
		return fmt.Errorf("invalid face: %s", face)
	}
	if orientation != model.OrientationRecovered && orientation != model.OrientationReversed && orientation != model.OrientationRested {
		return fmt.Errorf("invalid orientation: %s", orientation)
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	destIndex, err := playerIndex(state, controller)
	if err != nil {
		return err
	}
	if location.Zone == zone && state.Players[location.PlayerIndex].ID == controller {
		instance.Controller = controller
		instance.Face = face
		instance.Orientation = orientation
		if zone != model.ZoneServant {
			instance.GrantedDoubleCorrupt = false
			instance.AttackModifier = 0
			instance.DefenseModifier = 0
		}
		state.CardInstances[cardID] = instance
		return nil
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	dest, err := zoneCards(state, destIndex, zone)
	if err != nil {
		return err
	}
	instance.Controller = controller
	instance.Face = face
	instance.Orientation = orientation
	if zone != model.ZoneServant {
		instance.GrantedDoubleCorrupt = false
		instance.AttackModifier = 0
		instance.DefenseModifier = 0
	}
	*dest = append(*dest, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func SetOrientation(state *model.MatchState, cardID model.MatchCardID, orientation model.CardOrientation) error {
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	if orientation != model.OrientationRecovered && orientation != model.OrientationReversed && orientation != model.OrientationRested {
		return fmt.Errorf("invalid orientation: %s", orientation)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant && location.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not in a field zone that supports orientation", cardID)
	}
	instance.Orientation = orientation
	state.CardInstances[cardID] = instance
	return nil
}

// Switch flips Recovered↔Reversed. Rested cards are unchanged (CR 10.11).
func Switch(state *model.MatchState, cardID model.MatchCardID) error {
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
		return fmt.Errorf("card %q is not in a field zone that supports orientation", cardID)
	}
	switch instance.Orientation {
	case model.OrientationRecovered:
		instance.Orientation = model.OrientationReversed
	case model.OrientationReversed:
		instance.Orientation = model.OrientationRecovered
	case model.OrientationRested:
		// CR 10.11.1: nothing happens to a rested card.
	default:
		return fmt.Errorf("invalid orientation: %s", instance.Orientation)
	}
	state.CardInstances[cardID] = instance
	return nil
}

// SetFace sets face-up or face-down on a Servant or Caster zone card.
func SetFace(state *model.MatchState, cardID model.MatchCardID, face model.CardFace) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if face != model.CardFaceUp && face != model.CardFaceDown {
		return fmt.Errorf("invalid face: %s", face)
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
		return fmt.Errorf("card %q is not in a field zone that supports face", cardID)
	}
	instance.Face = face
	state.CardInstances[cardID] = instance
	return nil
}

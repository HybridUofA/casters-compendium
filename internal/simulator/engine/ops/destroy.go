package ops

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// DestroyServant moves a Servant-zone card (Servant or Barrier) to its owner's
// graveyard (CR 10.7). Barriers share the Servant zone in this engine.
func DestroyServant(state *model.MatchState, cardID model.MatchCardID) error {
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("destroyed servant %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate destroyed servant: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("destroyed servant %q is not in a servant zone", cardID)
	}
	zone := &state.Players[location.PlayerIndex].ServantZone
	*zone = slices.Delete(*zone, location.CardIndex, location.CardIndex+1)

	ownerIndex := -1
	for index, player := range state.Players {
		if player.ID == instance.Owner {
			ownerIndex = index
			break
		}
	}
	if ownerIndex == -1 {
		return fmt.Errorf("owner %q of destroyed servant was not found", instance.Owner)
	}
	instance.Controller = instance.Owner
	instance.Face = model.CardFaceUp
	instance.Orientation = model.OrientationRecovered
	instance.GrantedDoubleCorrupt = false
	instance.AttackModifier = 0
	instance.DefenseModifier = 0
	instance.EnteredOnTurn = 0
	instance.GrantedSlowStartUntilEOT = false
	instance.LosesSlowStartUntilEOT = false
	instance.BattledThisTurn = false
	state.Players[ownerIndex].Graveyard = append(state.Players[ownerIndex].Graveyard, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// MoveOrbToHand moves one Orb at orbIndex from playerID's Orbs zone into their hand.
// Battle timing / Break offers stay in engine; this is the leaf zone mutation.
func MoveOrbToHand(state *model.MatchState, playerID model.PlayerID, orbIndex int) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	orbs := state.Players[index].Orbs
	if orbIndex < 0 || orbIndex >= len(orbs) {
		return fmt.Errorf("orb index %d is out of range for %d orbs", orbIndex, len(orbs))
	}
	orbID := orbs[orbIndex]
	state.Players[index].Orbs = append(append([]model.MatchCardID(nil), orbs[:orbIndex]...), orbs[orbIndex+1:]...)
	state.Players[index].Hand = append(state.Players[index].Hand, orbID)
	if instance, ok := state.CardInstances[orbID]; ok {
		instance.Face = model.CardFaceDown
		state.CardInstances[orbID] = instance
	}
	return nil
}

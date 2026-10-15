package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// resolveExileReturnsAtEOT puts Ran Berfield-style exiled servants back onto
// their owner's field as Recovered servants.
func resolveExileReturnsAtEOT(state *model.MatchState) error {
	if state == nil || len(state.ExileReturnAtEOT) == 0 {
		return nil
	}
	pending := append([]model.MatchCardID(nil), state.ExileReturnAtEOT...)
	state.ExileReturnAtEOT = nil
	for _, cardID := range pending {
		instance, ok := state.CardInstances[cardID]
		if !ok {
			continue
		}
		owner := instance.Owner
		if owner == "" {
			owner = instance.Controller
		}
		if err := ops.PutAsServant(state, cardID, owner, model.CardFaceUp, model.OrientationRecovered); err != nil {
			return fmt.Errorf("return %q from exile: %w", cardID, err)
		}
	}
	return nil
}

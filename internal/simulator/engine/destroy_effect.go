package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// DestroyServant moves a Servant-zone card to its owner's graveyard and fires
// Last Words handlers (CR 11.7: put into graveyard from the field).
func DestroyServant(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
) error {
	return destroyServantWithCause(state, catalog, cardID, destroyCauseEffect, "")
}

func destroyServantWithCause(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
	cause destroyCause,
	destroyerController model.PlayerID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("destroyed servant %q does not exist", cardID)
	}
	noteDestroyed(state, catalog, instance, cause, destroyerController)
	if err := ops.DestroyServant(state, cardID); err != nil {
		return err
	}
	if err := triggerLastWordsEffects(state, catalog, cardID); err != nil {
		return fmt.Errorf("last words for %q: %w", cardID, err)
	}
	return nil
}

package engine

import (
	"fmt"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

const hildeWillowCardName = "Hilde Willow"

func isHildeWillow(definition gamecards.Card) bool {
	return strings.EqualFold(strings.TrimSpace(definition.Name), hildeWillowCardName) &&
		strings.EqualFold(strings.TrimSpace(definition.Type), "Caster")
}

// ActivateHildeWillow pays Rest via the shared Rest path and runs the Hilde
// recipe from master_effects (peek + PendingChoice leave/bottom). Returns the
// enemy player ID so callers can show a private peek overlay.
func ActivateHildeWillow(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) (enemyID model.PlayerID, err error) {
	if state == nil {
		return "", fmt.Errorf("state cannot be nil")
	}
	if catalog == nil {
		return "", fmt.Errorf("catalog cannot be nil")
	}
	RegisterDefaultEffectHandlers()

	actingIndex, err := findPlayerIndex(state, actingPlayerID)
	if err != nil {
		return "", err
	}
	enemyIndex := 1 - actingIndex
	enemyID = state.Players[enemyIndex].ID
	if len(state.Players[enemyIndex].Deck) == 0 {
		return "", fmt.Errorf("enemy deck is empty")
	}

	instance, ok := state.CardInstances[cardID]
	if !ok {
		return "", fmt.Errorf("caster %q not found", cardID)
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found || !isHildeWillow(definition) {
		return "", fmt.Errorf("%q is not Hilde Willow", cardID)
	}
	if !effects.HasMatch(&effectRegistry, effects.HookRest, instance.CardID, definition.Ability) {
		return "", fmt.Errorf("Hilde Willow has no registered Rest automation")
	}

	if err := ActivateRegisteredRest(state, catalog, actingPlayerID, cardID, expectedRevision); err != nil {
		return "", err
	}
	if state.PendingChoice.Kind == effects.PendingHildeDeckBottom && state.PendingChoice.Payload != "" {
		enemyID = model.PlayerID(state.PendingChoice.Payload)
	}
	return enemyID, nil
}

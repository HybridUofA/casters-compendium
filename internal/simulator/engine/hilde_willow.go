package engine

import (
	"fmt"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

const hildeWillowCardName = "Hilde Willow"

func isHildeWillow(definition gamecards.Card) bool {
	return strings.EqualFold(strings.TrimSpace(definition.Name), hildeWillowCardName) &&
		strings.EqualFold(strings.TrimSpace(definition.Type), "Caster")
}

// ActivateHildeWillow pays Hilde Willow's Rest cost and resolves the look
// immediately (activated-ability Chase resolution is not implemented yet).
// The caller shows the enemy deck-top privately and may then call
// MoveDeckTopToBottom. The acting player retains priority.
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
	if expectedRevision != state.Revision {
		return "", fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return "", fmt.Errorf("match is not in progress")
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != actingPlayerID {
		return "", fmt.Errorf("only the priority holder may activate Hilde Willow")
	}
	if state.PendingBreak.PlayerID != "" ||
		state.PendingDraw.Step != model.PendingDrawIdle ||
		state.PendingChoice.PlayerID != "" {
		return "", fmt.Errorf("cannot activate Hilde Willow while another decision is pending")
	}
	actingIndex, err := findPlayerIndex(state, actingPlayerID)
	if err != nil {
		return "", err
	}
	enemyIndex := 1 - actingIndex
	enemyID = state.Players[enemyIndex].ID
	if len(state.Players[enemyIndex].Deck) == 0 {
		return "", fmt.Errorf("enemy deck is empty")
	}

	foundInZone := false
	for _, matchID := range state.Players[actingIndex].CasterZone {
		if matchID == cardID {
			foundInZone = true
			break
		}
	}
	if !foundInZone {
		return "", fmt.Errorf("%q is not in your caster zone", cardID)
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return "", fmt.Errorf("caster %q not found", cardID)
	}
	if instance.Controller != actingPlayerID {
		return "", fmt.Errorf("you do not control %q", cardID)
	}
	if instance.Face != model.CardFaceUp {
		return "", fmt.Errorf("Hilde Willow must be face up")
	}
	if instance.Orientation != model.OrientationRecovered {
		return "", fmt.Errorf("Hilde Willow must be Recovered to Rest")
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found || !isHildeWillow(definition) {
		return "", fmt.Errorf("%q is not Hilde Willow", cardID)
	}

	instance.Orientation = model.OrientationRested
	state.CardInstances[cardID] = instance
	state.Revision++
	return enemyID, nil
}

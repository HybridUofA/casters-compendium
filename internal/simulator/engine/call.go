package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func CallFaceDownLevelOne(state *model.MatchState, actingPlayer model.PlayerID, cardID model.MatchCardID, expectedRevision model.Revision) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("revision %d does not match expected revision %d", state.Revision, expectedRevision)
	}
	if err := rules.ValidateFaceDownLevelOneCall(state, actingPlayer, cardID); err != nil {
		return fmt.Errorf("error calling caster: %w", err)
	}
	activeIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayer {
			activeIndex = index
		}
	}
	if activeIndex == -1 {
		return fmt.Errorf("player ID not found: %q", actingPlayer)
	}
	player := &state.Players[activeIndex]
	cardIndex := slices.Index(player.Hand, cardID)
	if cardIndex == -1 {
		return fmt.Errorf("card not found in hand: %q", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q not found in card instances", cardID)
	}
	instance.Face = model.CardFaceDown
	instance.Orientation = model.OrientationRecovered
	instance.Controller = actingPlayer
	state.CardInstances[cardID] = instance
	calledCard := state.Players[activeIndex].Hand[cardIndex]
	player.Hand = slices.Delete(player.Hand, cardIndex, cardIndex+1)
	player.CasterZone = append(player.CasterZone, calledCard)

	state.Turn.CallActionTaken = true
	state.Revision++
	return nil
}

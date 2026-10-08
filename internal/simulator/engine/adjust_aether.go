package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// AdjustAether applies a manual ±1 change to the acting player's own Aether pool.
func AdjustAether(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	command model.AdjustAetherCommand,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if err := rules.ValidateAdjustAether(state, actingPlayerID, command); err != nil {
		return fmt.Errorf("adjust aether: %w", err)
	}
	key, err := rules.ParseAetherElementKey(command.Element)
	if err != nil {
		return fmt.Errorf("adjust aether: %w", err)
	}
	playerIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("acting player ID not found")
	}
	rules.ApplyAetherPoolDelta(&state.Players[playerIndex].Aether, key, command.Delta)
	state.Revision++
	return nil
}

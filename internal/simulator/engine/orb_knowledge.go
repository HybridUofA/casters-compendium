package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// PeekOrb lets actingPlayer look at orbOwner's orb at orbIndex and remember it.
// orbIndex is 0-based into that player's Orbs zone.
func PeekOrb(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	orbOwnerID model.PlayerID,
	orbIndex int,
	expectedRevision model.Revision,
) (model.MatchCardID, error) {
	if state == nil {
		return "", fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return "", fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return "", fmt.Errorf("player ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return "", fmt.Errorf("match is not in progress")
	}
	if _, err := findPlayerIndex(state, actingPlayerID); err != nil {
		return "", err
	}
	ownerIndex, err := findPlayerIndex(state, orbOwnerID)
	if err != nil {
		return "", err
	}
	orbs := state.Players[ownerIndex].Orbs
	if orbIndex < 0 || orbIndex >= len(orbs) {
		return "", fmt.Errorf("orb index %d is out of range for %d orbs", orbIndex, len(orbs))
	}
	orbID := orbs[orbIndex]
	if _, ok := state.CardInstances[orbID]; !ok {
		return "", fmt.Errorf("orb %q does not exist", orbID)
	}
	model.MarkCardKnown(state, actingPlayerID, orbID)
	state.Revision++
	return orbID, nil
}

// RevealOrb shows actingPlayer's own orb at orbIndex to the opponent and lets
// them keep seeing its identity afterward.
func RevealOrb(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	orbIndex int,
	expectedRevision model.Revision,
) (model.MatchCardID, error) {
	if state == nil {
		return "", fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return "", fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return "", fmt.Errorf("player ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return "", fmt.Errorf("match is not in progress")
	}
	actingIndex, opponentIndex, err := battlePlayerIndexes(state, actingPlayerID)
	if err != nil {
		return "", err
	}
	orbs := state.Players[actingIndex].Orbs
	if orbIndex < 0 || orbIndex >= len(orbs) {
		return "", fmt.Errorf("orb index %d is out of range for %d orbs", orbIndex, len(orbs))
	}
	orbID := orbs[orbIndex]
	if _, ok := state.CardInstances[orbID]; !ok {
		return "", fmt.Errorf("orb %q does not exist", orbID)
	}
	opponentID := state.Players[opponentIndex].ID
	model.MarkCardKnown(state, opponentID, orbID)
	state.Revision++
	return orbID, nil
}

package rules

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateFaceDownLevelOneCall(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != model.PhaseCall {
		return fmt.Errorf("phase is %q, expected %q", state.Turn.Phase, model.PhaseCall)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("acting player %q is not the active player %q", actingPlayerID, state.Turn.ActivePlayer)
	}
	if state.Turn.CallActionTaken {
		return fmt.Errorf("call action has already been taken")
	}
	activeIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			activeIndex = index
		}
	}
	if activeIndex == -1 {
		return fmt.Errorf("acting player ID not found")
	}
	cardIndex := -1
	for index := range state.Players[activeIndex].Hand {
		if state.Players[activeIndex].Hand[index] == cardID {
			cardIndex = index
		}
	}
	if _, exists := state.CardInstances[cardID]; !exists {
		return fmt.Errorf("card ID %q not found in card instances", cardID)
	}
	if cardIndex == -1 {
		return fmt.Errorf("card not found in hand")
	}
	return nil
}

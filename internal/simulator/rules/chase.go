package rules

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateChasePriority(state *model.MatchState, actingPlayerID model.PlayerID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	actingPlayerIndex := -1
	priorityExists := false
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingPlayerIndex = index
		}
		if state.PriorityHolder == state.Players[index].ID {
			priorityExists = true
		}
	}
	if actingPlayerIndex == -1 {
		return fmt.Errorf("acting player ID not found")
	}
	if !priorityExists {
		return fmt.Errorf("neither player has priority")
	}
	if state.PriorityHolder != actingPlayerID {
		return fmt.Errorf("player %q does not hold priority; priority belongs to %q", actingPlayerID, state.PriorityHolder)
	}
	return nil
}

func ValidateAddChaseLink(state *model.MatchState, actingPlayerID model.PlayerID, sourceMatchCardID model.MatchCardID, kind model.ChaseLinkKind) error {
	err := ValidateChasePriority(state, actingPlayerID)
	if err != nil {
		return fmt.Errorf("error during priority validation: %w", err)
	}
	switch kind {
	case model.ChaseLinkCardPlay,
		model.ChaseLinkActivatedAbility,
		model.ChaseLinkTriggeredAbility:
	default:
		return fmt.Errorf("invalid Chase link kind %q", kind)
	}
	if strings.TrimSpace(string(sourceMatchCardID)) == "" {
		return fmt.Errorf("source card ID must not be empty")
	}
	source, exists := state.CardInstances[sourceMatchCardID]
	if !exists {
		return fmt.Errorf("source card %q was not found", sourceMatchCardID)
	}
	if source.Controller != actingPlayerID {
		return fmt.Errorf("card's controller %q is not the player activating the card %q", source.Controller, actingPlayerID)
	}
	if state.NextLinkID == 0 {
		return fmt.Errorf("next link ID must be positive; is %d", state.NextLinkID)
	}
	return nil
}

func ValidatePassPriority(state *model.MatchState, actingPlayerID model.PlayerID) error {
	err := ValidateChasePriority(state, actingPlayerID)
	if err != nil {
		return fmt.Errorf("error during priority validation: %w", err)
	}
	if state.PassCount < 0 || state.PassCount > 1 {
		return fmt.Errorf("pass count is %d, must be between 0 and 1", state.PassCount)
	}
	return nil
}

func ValidateResolveTopChaseLink(state *model.MatchState, actingPlayerID model.PlayerID) error {
	err := ValidateChasePriority(state, actingPlayerID)
	if err != nil {
		return fmt.Errorf("error during priority validation: %w", err)
	}
	if state.PassCount != 1 {
		return fmt.Errorf("pass count must be 1, is %d", state.PassCount)
	}
	if len(state.ChaseLinks) <= 0 {
		return fmt.Errorf("an empty chase cannot resolve a link")
	}
	return nil
}

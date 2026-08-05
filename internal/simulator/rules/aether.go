package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateGenerateNonElementalAether(
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
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
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
	if exists := slices.Contains(state.Players[activeIndex].CasterZone, cardID); !exists {
		return fmt.Errorf("card must exist in caster zone: %q", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q does not exist in card instances", cardID)
	}
	if instance.Face != model.CardFaceDown {
		return fmt.Errorf("cannot generate non-elemental Aether: card is face up")
	}
	if instance.Orientation != model.OrientationRecovered {
		return fmt.Errorf("cannot rest an already rested caster")
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("cannot rest casters you do not control")
	}
	return nil
}

func ValidateUseCasterToken(state *model.MatchState, actingPlayerID model.PlayerID, tokenID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match status is %q, must be %q", state.MatchStatus, model.StatusInProgress)
	}
	actingIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingIndex = index
		}
	}
	if actingIndex == -1 {
		return fmt.Errorf("player %q not found in players", actingPlayerID)
	}
	player := &state.Players[actingIndex]
	cardIndex := slices.Index(player.CasterZone, tokenID)
	if cardIndex == -1 {
		return fmt.Errorf("caster token not found in caster zone")
	}
	instance, exists := state.CardInstances[tokenID]
	if !exists {
		return fmt.Errorf("caster token %q not found", tokenID)
	}
	if instance.CardID != model.CasterTokenCardID {
		return fmt.Errorf("card %q is not a caster token", tokenID)
	}
	if instance.CardCategory != model.CategoryTokenCard {
		return fmt.Errorf("card %q is not a token", tokenID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("owner %q is not active player %q", instance.Owner, actingPlayerID)
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("controller %q is not active player %q", instance.Controller, actingPlayerID)
	}
	if instance.Orientation != model.OrientationRecovered {
		return fmt.Errorf("caster token must be recovered")
	}
	if instance.Face != model.CardFaceUp {
		return fmt.Errorf("caster token must be face-up")
	}
	return nil
}

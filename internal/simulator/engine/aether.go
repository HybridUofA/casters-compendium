package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func GenerateNonElementalAether(
	state *model.MatchState,
	actingPlayer model.PlayerID,
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("revision %d does not match expected revision %d", state.Revision, expectedRevision)
	}
	if err := rules.ValidateGenerateNonElementalAether(state, actingPlayer, cardID); err != nil {
		return fmt.Errorf("error generating non-elemental aether: %w", err)
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
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q does not exist in instances", cardID)
	}
	instance.Orientation = model.OrientationRested
	state.CardInstances[cardID] = instance
	player.Aether.NonElemental++
	state.Revision++
	return nil
}

func UseCasterToken(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	tokenID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("revision %d does not match expected revision %d", state.Revision, expectedRevision)
	}
	if err := rules.ValidateUseCasterToken(state, actingPlayerID, tokenID); err != nil {
		return fmt.Errorf("error generating non-elemental aether from caster token: %w", err)
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
	_, exists := state.CardInstances[tokenID]
	if !exists {
		return fmt.Errorf("caster token %q not found", tokenID)
	}
	delete(state.CardInstances, tokenID)
	player.CasterZone = slices.Delete(player.CasterZone, cardIndex, cardIndex+1)
	player.Aether.NonElemental++
	state.Revision++
	return nil
}

func GenerateCasterAether(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("revision %d does not match expected revision %d", state.Revision, expectedRevision)
	}
	element, amount, err := rules.ValidateGenerateCasterAether(state, catalog, actingPlayerID, cardID)
	if err != nil {
		return fmt.Errorf("error generating aether: %w", err)
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
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q does not exist in instances", cardID)
	}

	player := &state.Players[actingIndex]
	pool := &player.Aether
	var destination *int
	switch element {
	case model.ElementAes:
		destination = &pool.Aes
	case model.ElementAqua:
		destination = &pool.Aqua
	case model.ElementIgnus:
		destination = &pool.Ignus
	case model.ElementLuna:
		destination = &pool.Luna
	case model.ElementSilva:
		destination = &pool.Silva
	case model.ElementSolis:
		destination = &pool.Solis
	case model.ElementTerra:
		destination = &pool.Terra
	case model.ElementVoid:
		destination = &pool.Void
	default:
		return fmt.Errorf("unsupported Caster element %q", element)
	}
	instance.Orientation = model.OrientationRested
	state.CardInstances[cardID] = instance
	*destination += amount
	state.Revision++
	return nil
}

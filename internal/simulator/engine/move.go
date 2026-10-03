package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func playerZoneCards(player *model.PlayerState, zone model.Zone) (*[]model.MatchCardID, error) {
	if player == nil {
		return nil, fmt.Errorf("player cannot be nil")
	}
	switch zone {
	case model.ZoneHand:
		return &player.Hand, nil
	case model.ZoneDeck:
		return &player.Deck, nil
	case model.ZoneOrbs:
		return &player.Orbs, nil
	case model.ZoneCaster:
		return &player.CasterZone, nil
	case model.ZoneServant:
		return &player.ServantZone, nil
	case model.ZoneGraveyard:
		return &player.Graveyard, nil
	case model.ZoneExile:
		return &player.Exile, nil
	default:
		return nil, fmt.Errorf("unsupported zone: %q", zone)
	}
}

func MoveCard(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, command model.MoveCardCommand, expectedRevision model.Revision) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if err := rules.ValidateMoveCard(state, catalog, actingPlayerID, command); err != nil {
		return fmt.Errorf("error during validation: %w", err)
	}
	location, err := model.FindCardLocation(state, command.CardID)
	if err != nil {
		return fmt.Errorf("error during finding card location: %w", err)
	}
	destPlayerIndex := -1
	for index, player := range state.Players {
		if player.ID == command.DestinationPlayerID {
			destPlayerIndex = index
			break
		}
	}
	if destPlayerIndex == -1 {
		return fmt.Errorf("player index for %q not found", command.DestinationPlayerID)
	}
	sourceSlice, err := playerZoneCards(&state.Players[location.PlayerIndex], location.Zone)
	if err != nil {
		return fmt.Errorf("error resolving source slice: %w", err)
	}
	destSlice, err := playerZoneCards(&state.Players[destPlayerIndex], command.DestinationZone)
	if err != nil {
		return fmt.Errorf("error resolving destination slice: %w", err)
	}
	instance, exists := state.CardInstances[command.CardID]
	if !exists {
		return fmt.Errorf("card %q does not exist", command.CardID)
	}
	instance.Controller = command.DestinationPlayerID
	instance.Face = command.DestinationFace
	instance.Orientation = command.EntryOrientation
	insertionIndex := len(*destSlice)
	switch command.DestinationZone {
	case model.ZoneDeck:
		if command.Placement == model.DeckPlacementTop {
			insertionIndex = 0
		}
	case model.ZoneOrbs:
		return fmt.Errorf("orb insertion not implemented yet")
	}
	*sourceSlice = slices.Delete(
		*sourceSlice,
		location.CardIndex,
		location.CardIndex+1,
	)
	*destSlice = slices.Insert(
		*destSlice,
		insertionIndex,
		command.CardID,
	)
	state.CardInstances[command.CardID] = instance
	state.Revision++
	return nil
}

package ops

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func playerIndex(state *model.MatchState, playerID model.PlayerID) (int, error) {
	if state == nil {
		return -1, fmt.Errorf("state cannot be nil")
	}
	if strings.TrimSpace(string(playerID)) == "" {
		return -1, fmt.Errorf("player ID cannot be empty")
	}
	for index, player := range state.Players {
		if player.ID == playerID {
			return index, nil
		}
	}
	return -1, fmt.Errorf("player not found")
}

func zoneCards(state *model.MatchState, playerIndex int, zone model.Zone) (*[]model.MatchCardID, error) {
	switch zone {
	case model.ZoneHand:
		return &state.Players[playerIndex].Hand, nil
	case model.ZoneDeck:
		return &state.Players[playerIndex].Deck, nil
	case model.ZoneOrbs:
		return &state.Players[playerIndex].Orbs, nil
	case model.ZoneCaster:
		return &state.Players[playerIndex].CasterZone, nil
	case model.ZoneServant:
		return &state.Players[playerIndex].ServantZone, nil
	case model.ZoneGraveyard:
		return &state.Players[playerIndex].Graveyard, nil
	case model.ZoneExile:
		return &state.Players[playerIndex].Exile, nil
	default:
		return nil, fmt.Errorf("unsupported zone %q", zone)
	}
}

func removeCardAt(location model.CardLocation, state *model.MatchState) error {
	zone, err := zoneCards(state, location.PlayerIndex, location.Zone)
	if err != nil {
		return err
	}
	if location.CardIndex < 0 || location.CardIndex >= len(*zone) {
		return fmt.Errorf("card index %d out of range for zone %q", location.CardIndex, location.Zone)
	}
	*zone = slices.Delete(*zone, location.CardIndex, location.CardIndex+1)
	return nil
}

func clearFieldStatus(instance *model.CardInstance) {
	instance.Controller = instance.Owner
	instance.Face = model.CardFaceUp
	instance.Orientation = model.OrientationRecovered
	instance.GrantedDoubleCorrupt = false
	instance.AttackModifier = 0
	instance.DefenseModifier = 0
	instance.EnteredOnTurn = 0
	instance.GrantedSlowStartUntilEOT = false
	instance.LosesSlowStartUntilEOT = false
	instance.GrantedVoidRestATK500UntilEOT = false
	instance.BattledThisTurn = false
}

func takeTopCards(deck []model.MatchCardID, count int) (taken []model.MatchCardID, remaining []model.MatchCardID, err error) {
	if count < 0 {
		return nil, nil, fmt.Errorf("count cannot be negative: %d", count)
	}
	if count > len(deck) {
		return nil, nil, fmt.Errorf("count cannot be greater than amount of cards in deck (%d): %d", len(deck), count)
	}
	taken = slices.Clone(deck[:count])
	remaining = slices.Clone(deck[count:])
	return taken, remaining, nil
}

// findStockHost returns the field card that lists stockCardID in its Stock.
// Stocked cards are not present in zone slices, so FindCardLocation cannot see them.
func findStockHost(state *model.MatchState, stockCardID model.MatchCardID) (hostID model.MatchCardID, stockIndex int, err error) {
	if state == nil {
		return "", -1, fmt.Errorf("state cannot be nil")
	}
	found := false
	for id, instance := range state.CardInstances {
		for index, stockID := range instance.Stock {
			if stockID != stockCardID {
				continue
			}
			if found {
				return "", -1, fmt.Errorf("stock card %q is attached under multiple hosts", stockCardID)
			}
			found = true
			hostID = id
			stockIndex = index
		}
	}
	if !found {
		return "", -1, fmt.Errorf("stock card %q is not attached under any host", stockCardID)
	}
	return hostID, stockIndex, nil
}

func appendToOwnerGraveyard(state *model.MatchState, cardID model.MatchCardID, instance *model.CardInstance) error {
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(instance)
	state.Players[ownerIndex].Graveyard = append(state.Players[ownerIndex].Graveyard, cardID)
	state.CardInstances[cardID] = *instance
	return nil
}

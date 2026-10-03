package model

import (
	"fmt"
	"strings"
)

func FindCardLocation(
	state *MatchState,
	cardID MatchCardID,
) (CardLocation, error) {
	if state == nil {
		return CardLocation{}, fmt.Errorf("state cannot be nil")
	}
	if strings.TrimSpace(string(cardID)) == "" {
		return CardLocation{}, fmt.Errorf("card ID cannot be blank")
	}
	location := CardLocation{}
	found := false
	for pindex, player := range state.Players {
		zones := []struct {
			zone  Zone
			cards []MatchCardID
		}{
			{zone: ZoneHand, cards: player.Hand},
			{zone: ZoneDeck, cards: player.Deck},
			{zone: ZoneOrbs, cards: player.Orbs},
			{zone: ZoneCaster, cards: player.CasterZone},
			{zone: ZoneServant, cards: player.ServantZone},
			{zone: ZoneGraveyard, cards: player.Graveyard},
			{zone: ZoneExile, cards: player.Exile},
		}
		for _, slice := range zones {
			for cindex, card := range slice.cards {
				if card == cardID {
					if found {
						return CardLocation{}, fmt.Errorf("card %q already found", cardID)
					}
					found = true
					location = CardLocation{pindex, slice.zone, cindex}
				}
			}
		}
	}
	if !found {
		return CardLocation{}, fmt.Errorf("card %q not found", cardID)
	}
	return location, nil
}

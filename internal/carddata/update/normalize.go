// Package cardupdate normalizes records retrieved from external card-data sources.
package cardupdate

import (
	"fmt"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

// FromSpeedrobo maps a remote card-detail record into the application's normalized Card model.
func FromSpeedrobo(detail speedrobo.CardDetail) (gamecards.Card, error) {
	var isPlaytesting bool
	switch detail.IsPlaytesting {
	case "0":
		isPlaytesting = false
	case "1":
		isPlaytesting = true
	default:
		return gamecards.Card{}, fmt.Errorf(
			"card %q has unexpected playtesting value %q",
			detail.CardKey,
			detail.IsPlaytesting,
		)
	}

	card := gamecards.Card{
		ID:            detail.ID,
		Name:          detail.CardKey,
		ImageURL:      detail.ImageURL,
		Expansion:     detail.ExpansionName,
		IsPlaytesting: isPlaytesting,
		ExtraFields:   make(map[string]string),
	}

	for _, field := range detail.Fields {
		switch field.Label {
		case "Name":
			card.Name = field.Value
		case "Subname":
			card.Subname = field.Value
		case "Type":
			card.Type = field.Value
		case "Element":
			card.Element = field.Value
		case "Traits":
			card.Traits = field.Value
		case "CostLevel", "Cost/Lv":
			card.CostLevel = field.Value
		case "Attack", "ATK":
			card.Attack = field.Value
		case "Defense", "DEF":
			card.Defense = field.Value
		case "Ability":
			card.Ability = field.Value
		case "Flavor":
			card.Flavor = field.Value
		case "Artist":
			card.Artist = field.Value
		case "CardNumber", "Card Number":
			card.CardNumber = field.Value
		case "Count":
			card.Count = field.Value
		case "ImageURL":
			card.ImageURL = field.Value
		case "Expansion":
			card.Expansion = field.Value
		default:
			card.ExtraFields[field.Label] = field.Value
		}
	}

	if len(card.ExtraFields) == 0 {
		card.ExtraFields = nil
	}

	return card, nil
}

package rules

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ResolveCardDefinition(catalog CardCatalog, cardID model.CardID) (cards.Card, error) {
	if catalog == nil {
		return cards.Card{}, fmt.Errorf("catalog cannot be nil")
	}
	if strings.TrimSpace(string(cardID)) == "" {
		return cards.Card{}, fmt.Errorf("cardID cannot be empty")
	}
	card, found := catalog.FindByID(string(cardID))
	if !found {
		return cards.Card{}, fmt.Errorf("error looking up card %q", cardID)
	}
	return card, nil
}

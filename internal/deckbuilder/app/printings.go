package deckbuilder

import (
	"fmt"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

// printingOptionLabel identifies a physical printing without repeating the
// card's rules name, which is already displayed above the selector.
func printingOptionLabel(card cards.Card) string {
	parts := make([]string, 0, 3)
	if cardNumber := strings.TrimSpace(card.CardNumber); cardNumber != "" {
		parts = append(parts, cardNumber)
	}
	if expansion := strings.TrimSpace(card.Expansion); expansion != "" {
		parts = append(parts, expansion)
	}

	for key, value := range card.ExtraFields {
		if strings.EqualFold(strings.TrimSpace(key), "rarity") {
			if rarity := strings.TrimSpace(value); rarity != "" {
				parts = append(parts, rarity)
			}
			break
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Printing %s", strings.TrimSpace(card.ID))
	}

	return strings.Join(parts, " · ")
}

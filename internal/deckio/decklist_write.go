package deckio

import (
	"fmt"
	"io"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// WriteDeckList emits the human-readable main- and side-deck interchange format.
func WriteDeckList(writer io.Writer, deck *decks.Deck, repository decks.CardCatalog) error {
	if _, err := fmt.Fprintf(writer, "Deck: %s\n", deck.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Game: The Caster Chronicles\n\n"); err != nil {
		return err
	}
	for _, entry := range deck.MainDeck {
		card, found := repository.FindByID(entry.CardID)
		if !found {
			return fmt.Errorf("main deck contains unknown card ID %q", entry.CardID)
		}
		if _, err := fmt.Fprintf(writer, "%dx %s (%s)\n", entry.Quantity, card.Name, card.Expansion); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "\nTotal: %d cards\n", deck.MainTotal()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "\n--- Side Deck ---\n\n"); err != nil {
		return err
	}
	for _, entry := range deck.SideDeck {
		card, found := repository.FindByID(entry.CardID)
		if !found {
			return fmt.Errorf("side deck contains unknown card ID %q", entry.CardID)
		}
		if _, err := fmt.Fprintf(writer, "%dx %s (%s)\n", entry.Quantity, card.Name, card.Expansion); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "\nSide Total: %d cards\n", deck.SideTotal()); err != nil {
		return err
	}
	return nil
}

package decks

import (
	"fmt"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"io"
	"strings"
)

func NewDeck(name string) (*Deck, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return nil, fmt.Errorf("deck name cannot be empty")
	}

	return &Deck{
		SchemaVersion: 1,
		Name:          name,
		MainDeck:      make([]DeckEntry, 0),
		SideDeck:      make([]DeckEntry, 0),
	}, nil
}

type Zone string

const (
	MainZone Zone = "main"
	SideZone Zone = "side"
)

func (deck *Deck) QuantityOf(cardID string) int {
	cardID = strings.TrimSpace(cardID)

	return quantityIn(deck.MainDeck, cardID) +
		quantityIn(deck.SideDeck, cardID)
}

func (deck *Deck) QuantityInZone(
	zone Zone,
	cardID string,
) (int, error) {
	entries, err := deck.entriesFor(zone)
	if err != nil {
		return 0, err
	}

	cardID = strings.TrimSpace(cardID)

	return quantityIn(*entries, cardID), nil
}

func (deck *Deck) AddCard(
	zone Zone,
	cardID string,
	quantity int,
) error {
	cardID = strings.TrimSpace(cardID)

	if cardID == "" {
		return fmt.Errorf("card ID cannot be empty")
	}

	if quantity <= 0 {
		return fmt.Errorf(
			"quantity must be positive, received %d",
			quantity,
		)
	}

	entries, err := deck.entriesFor(zone)
	if err != nil {
		return err
	}

	for index := range *entries {
		if (*entries)[index].CardID == cardID {
			(*entries)[index].Quantity += quantity
			return nil
		}
	}

	*entries = append(*entries, DeckEntry{
		CardID:   cardID,
		Quantity: quantity,
	})

	return nil
}

func (deck *Deck) SetQuantity(
	zone Zone,
	cardID string,
	quantity int,
) error {
	cardID = strings.TrimSpace(cardID)

	if cardID == "" {
		return fmt.Errorf("card ID cannot be empty")
	}

	if quantity < 0 {
		return fmt.Errorf(
			"quantity cannot be negative, received %d",
			quantity,
		)
	}

	entries, err := deck.entriesFor(zone)
	if err != nil {
		return err
	}

	for index := range *entries {
		if (*entries)[index].CardID != cardID {
			continue
		}

		if quantity == 0 {
			*entries = append(
				(*entries)[:index],
				(*entries)[index+1:]...,
			)
		} else {
			(*entries)[index].Quantity = quantity
		}

		return nil
	}

	if quantity > 0 {
		*entries = append(*entries, DeckEntry{
			CardID:   cardID,
			Quantity: quantity,
		})
	}

	return nil
}

func (deck *Deck) RemoveCard(
	zone Zone,
	cardID string,
	quantity int,
) error {
	cardID = strings.TrimSpace(cardID)

	if quantity <= 0 {
		return fmt.Errorf(
			"quantity must be positive, received %d",
			quantity,
		)
	}

	currentQuantity, err := deck.QuantityInZone(zone, cardID)
	if err != nil {
		return err
	}

	if currentQuantity == 0 {
		return fmt.Errorf(
			"card ID %q is not in the %s deck",
			cardID,
			zone,
		)
	}

	if quantity > currentQuantity {
		return fmt.Errorf(
			"cannot remove %d copies of card %q; %s deck contains %d",
			quantity,
			cardID,
			zone,
			currentQuantity,
		)
	}

	return deck.SetQuantity(
		zone,
		cardID,
		currentQuantity-quantity,
	)
}

func totalCards(entries []DeckEntry) int {
	total := 0

	for _, entry := range entries {
		total += entry.Quantity
	}

	return total
}

func (deck *Deck) MainTotal() int {
	return totalCards(deck.MainDeck)
}

func (deck *Deck) SideTotal() int {
	return totalCards(deck.SideDeck)
}

func (deck *Deck) TotalCards() int {
	return deck.MainTotal() + deck.SideTotal()
}

func (deck *Deck) entriesFor(zone Zone) (*[]DeckEntry, error) {
	switch zone {
	case MainZone:
		return &deck.MainDeck, nil
	case SideZone:
		return &deck.SideDeck, nil
	default:
		return nil, fmt.Errorf("unknown deck zone %q", zone)
	}
}

func quantityIn(entries []DeckEntry, cardID string) int {
	for _, entry := range entries {
		if entry.CardID == cardID {
			return entry.Quantity
		}
	}

	return 0
}

func WriteDeckList(
	writer io.Writer,
	deck *Deck,
	repository *cards.Repository,
) error {
	fmt.Fprintf(writer, "Deck Name: %s\n\n", deck.Name)
	fmt.Fprintf(writer, "Main Deck (%d)\n", deck.MainTotal())
	for _, entry := range deck.MainDeck {
		cardData, _ := repository.FindByID(entry.CardID)
		fmt.Fprintf(writer, "%dx %s [%s]\n", entry.Quantity, cardData.Name, cardData.Expansion)
	}
	fmt.Fprintf(writer, "\nSide Deck (%d)\n", deck.SideTotal())
	for _, entry := range deck.SideDeck {
		cardData, _ := repository.FindByID(entry.CardID)
		fmt.Fprintf(writer, "%dx %s [%s]\n", entry.Quantity, cardData.Name, cardData.Expansion)
	}
	return nil
}

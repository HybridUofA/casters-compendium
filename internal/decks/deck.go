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
	MaxCopiesPerCard = 4
	MaxMainDeckCards = 50
	MaxSideDeckCards = 12
)

func normalizeCardName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (deck *Deck) CopiesOfCard(
	target cards.Card,
	repository *cards.Repository,
) int {
	total := 0
	targetName := normalizeCardName(target.Name)

	countEntries := func(entries []DeckEntry) {
		for _, entry := range entries {
			// Exact database record match.
			if entry.CardID == target.ID {
				total += entry.Quantity
				continue
			}

			// Check alternate printings by resolving their IDs.
			entryCard, found := repository.FindByID(
				entry.CardID,
			)
			if !found {
				continue
			}

			if normalizeCardName(entryCard.Name) ==
				targetName {
				total += entry.Quantity
			}
		}
	}

	countEntries(deck.MainDeck)
	countEntries(deck.SideDeck)

	return total
}

func (deck *Deck) ValidateCopyLimits() error {
	copyCounts := make(map[string]int)

	for _, entry := range deck.MainDeck {
		copyCounts[entry.CardID] += entry.Quantity
	}

	for _, entry := range deck.SideDeck {
		copyCounts[entry.CardID] += entry.Quantity
	}

	for cardID, quantity := range copyCounts {
		if quantity > MaxCopiesPerCard {
			return fmt.Errorf("card %s has %d copies; maximum is 4", cardID, quantity)
		}
	}

	return nil
}

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

func (deck *Deck) AddCardChecked(
	zone Zone,
	card cards.Card,
	quantity int,
	repository *cards.Repository,
) (bool, error) {
	deck.EnsureOrder()
	if quantity <= 0 {
		return false, fmt.Errorf(
			"quantity must be greater than zero",
		)
	}

	currentCopies := deck.CopiesOfCard(
		card,
		repository,
	)

	remainingCopies := MaxCopiesPerCard - currentCopies

	// Already at the limit: silently do nothing.
	if remainingCopies <= 0 {
		return false, nil
	}

	var zoneSpace int
	switch zone {
	case MainZone:
		zoneSpace = MaxMainDeckCards - deck.MainTotal()
	case SideZone:
		zoneSpace = MaxSideDeckCards - deck.SideTotal()
	default:
		return false, fmt.Errorf("unknown deck zone %q", zone)
	}

	if zoneSpace <= 0 {
		return false, nil
	}
	// Prevent bulk additions from exceeding the limit.
	if quantity > remainingCopies {
		quantity = remainingCopies
	}

	err := deck.AddCard(
		zone,
		card.ID,
		quantity,
	)
	if err != nil {
		return false, err
	}

	order, err := deck.orderForZone(zone)
	if err != nil {
		return false, err
	}

	for copyNumber := 0;
	copyNumber < quantity;
	copyNumber++ {
		*order = append(*order, card.ID)
	}

	return true, nil
}

func (deck *Deck) RemoveCardAt(
	zone Zone,
	index int,
) error {
	deck.EnsureOrder()

	order, err := deck.orderForZone(zone)
	if err != nil {
		return err
	}

	if index < 0 || index >= len(*order) {
		return fmt.Errorf(
			"card index %d is outside the deck",
			index,
		)
	}

	cardID := (*order)[index]

	if err := deck.RemoveCard(
		zone,
		cardID,
		1,
	); err != nil {
		return err
	}

	*order = append(
		(*order)[:index],
		(*order)[index+1:]...,
	)

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

func (deck *Deck) EnsureOrder() {
	if len(deck.MainOrder) != deck.MainTotal() {
		deck.MainOrder = make(
			[]string,
			0,
			deck.MainTotal(),
		)

		for _, entry := range deck.MainDeck {
			for copyNumber := 0;
				copyNumber < entry.Quantity;
				copyNumber++ {
				deck.MainOrder = append(
					deck.MainOrder,
					entry.CardID,
				)
			}
		}
	}

	if len(deck.SideOrder) != deck.SideTotal() {
		deck.SideOrder = make(
			[]string,
			0,
			deck.SideTotal(),
		)

		for _, entry := range deck.SideDeck {
			for copyNumber := 0;
				copyNumber < entry.Quantity;
				copyNumber++ {
				deck.SideOrder = append(
					deck.SideOrder,
					entry.CardID,
				)
			}
		}
	}
}

func (deck *Deck) orderForZone(
	zone Zone,
) (*[]string, error) {
	switch zone {
	case MainZone:
		return &deck.MainOrder, nil

	case SideZone:
		return &deck.SideOrder, nil

	default:
		return nil, fmt.Errorf(
			"unknown deck zone: %v",
			zone,
		)
	}
}

func insertCardID(
	cardIDs []string,
	index int,
	cardID string,
) []string {
	if index < 0 {
		index = 0
	}

	if index > len(cardIDs) {
		index = len(cardIDs)
	}

	cardIDs = append(cardIDs, "")

	copy(
		cardIDs[index+1:],
		cardIDs[index:],
	)

	cardIDs[index] = cardID

	return cardIDs
}

func (deck *Deck) MoveOrderedCard(
	fromZone Zone,
	fromIndex int,
	toZone Zone,
	toIndex int,
) (bool, error) {
	deck.EnsureOrder()

	source, err := deck.orderForZone(fromZone)
	if err != nil {
		return false, err
	}

	destination, err := deck.orderForZone(toZone)
	if err != nil {
		return false, err
	}

	if fromIndex < 0 ||
		fromIndex >= len(*source) {
		return false, fmt.Errorf(
			"source index %d is outside the deck",
			fromIndex,
		)
	}

	/*
		Reordering within the same zone.
	*/
	if fromZone == toZone {
		cardID := (*source)[fromIndex]

		updated := append(
			(*source)[:fromIndex],
			(*source)[fromIndex+1:]...,
		)

		// The removal shifted later positions left.
		if toIndex > fromIndex {
			toIndex--
		}

		updated = insertCardID(
			updated,
			toIndex,
			cardID,
		)

		*source = updated

		return true, nil
	}

	/*
		Moving between zones.
	*/
	switch toZone {
	case MainZone:
		if deck.MainTotal() >= MaxMainDeckCards {
			return false, nil
		}

	case SideZone:
		if deck.SideTotal() >= MaxSideDeckCards {
			return false, nil
		}
	}

	cardID := (*source)[fromIndex]

	if err := deck.RemoveCard(
		fromZone,
		cardID,
		1,
	); err != nil {
		return false, err
	}

	if err := deck.AddCard(
		toZone,
		cardID,
		1,
	); err != nil {
		// Restore the source zone if the move fails.
		_ = deck.AddCard(
			fromZone,
			cardID,
			1,
		)

		return false, err
	}

	*source = append(
		(*source)[:fromIndex],
		(*source)[fromIndex+1:]...,
	)

	*destination = insertCardID(
		*destination,
		toIndex,
		cardID,
	)

	return true, nil
}
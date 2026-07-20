package decks

import (
	"fmt"
	"slices"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

// CardCatalog supplies the card lookups required by deck rules and ordering.
type CardCatalog interface {
	FindByID(id string) (gamecards.Card, bool)
}

// NewDeck creates an empty versioned deck with a nonblank name.
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
	MainZone         Zone = "main"
	SideZone         Zone = "side"
	MaxCopiesPerCard      = 4
	MaxMainDeckCards      = 50
	MaxSideDeckCards      = 12
)

// normalizeCardName prepares names for copy-limit comparisons across alternate printings.
func normalizeCardName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// CopiesOfCard counts a card across both zones, including alternate printings with the same name.
func (deck *Deck) CopiesOfCard(
	target gamecards.Card,
	repository CardCatalog,
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

// ValidateCopyLimits verifies that no exact card identifier exceeds the configured copy limit.
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

// QuantityOf returns the number of copies of cardID across the complete deck.
func (deck *Deck) QuantityOf(cardID string) int {
	cardID = strings.TrimSpace(cardID)

	return quantityIn(deck.MainDeck, cardID) +
		quantityIn(deck.SideDeck, cardID)
}

// QuantityInZone returns the number of copies of cardID in one deck zone.
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

// AddCard adjusts aggregate zone entries without applying deck-building limits.
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

// AddCardChecked adds legal copies to the end of a zone's visible order.
func (deck *Deck) AddCardChecked(
	zone Zone,
	card gamecards.Card,
	quantity int,
	repository CardCatalog,
) (bool, error) {
	return deck.AddCardCheckedAt(
		zone,
		card,
		quantity,
		repository,
		-1,
	)
}

// AddCardCheckedAt applies copy and zone limits before inserting copies at a visible position.
func (deck *Deck) AddCardCheckedAt(
	zone Zone,
	card gamecards.Card,
	quantity int,
	repository CardCatalog,
	index int,
) (bool, error) {
	if quantity <= 0 {
		return false, fmt.Errorf("quantity must be greater than zero")
	}
	deck.EnsureOrder()
	currentCopies := deck.CopiesOfCard(card, repository)
	copySpace := MaxCopiesPerCard - currentCopies
	if copySpace <= 0 {
		return false, nil
	}
	var zoneSpace int
	switch zone {
	case MainZone:
		zoneSpace = MaxMainDeckCards - deck.MainTotal()
	case SideZone:
		zoneSpace = MaxSideDeckCards - deck.SideTotal()
	default:
		return false, fmt.Errorf("unknown deck zone %v", zone)
	}

	if zoneSpace <= 0 {
		return false, nil
	}

	allowedQuantity := quantity
	if allowedQuantity > copySpace {
		allowedQuantity = copySpace
	}
	if allowedQuantity > zoneSpace {
		allowedQuantity = zoneSpace
	}
	if err := deck.AddCard(zone, card.ID, allowedQuantity); err != nil {
		return false, err
	}

	order, err := deck.orderForZone(zone)
	if err != nil {
		return false, err
	}
	if index < 0 || index > len(*order) {
		index = len(*order)
	}
	for copyNumber := 0; copyNumber < allowedQuantity; copyNumber++ {
		*order = insertCardID(*order, index+copyNumber, card.ID)
	}
	return true, nil
}

// RemoveCardAt removes the physical card copy at a zone's visible index.
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

// SetQuantity replaces the aggregate quantity for one card in a zone.
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

// RemoveCard subtracts copies from an aggregate zone entry after validating availability.
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

// totalCards sums aggregate quantities in a zone.
func totalCards(entries []DeckEntry) int {
	total := 0

	for _, entry := range entries {
		total += entry.Quantity
	}

	return total
}

// MainTotal returns the physical card count in the main deck.
func (deck *Deck) MainTotal() int {
	return totalCards(deck.MainDeck)
}

// SideTotal returns the physical card count in the side deck.
func (deck *Deck) SideTotal() int {
	return totalCards(deck.SideDeck)
}

// TotalCards returns the combined physical card count of both zones.
func (deck *Deck) TotalCards() int {
	return deck.MainTotal() + deck.SideTotal()
}

// entriesFor resolves a zone to its mutable aggregate entry slice.
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

// quantityIn returns one card's aggregate quantity from a zone entry slice.
func quantityIn(entries []DeckEntry, cardID string) int {
	for _, entry := range entries {
		if entry.CardID == cardID {
			return entry.Quantity
		}
	}

	return 0
}

// EnsureOrder reconstructs per-copy visual order when it is absent or inconsistent in length.
func (deck *Deck) EnsureOrder() {
	if len(deck.MainOrder) != deck.MainTotal() {
		deck.MainOrder = make(
			[]string,
			0,
			deck.MainTotal(),
		)

		for _, entry := range deck.MainDeck {
			for copyNumber := 0; copyNumber < entry.Quantity; copyNumber++ {
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
			for copyNumber := 0; copyNumber < entry.Quantity; copyNumber++ {
				deck.SideOrder = append(
					deck.SideOrder,
					entry.CardID,
				)
			}
		}
	}
}

// Sort orders both deck zones by type, cost, name, card number, and identifier.
func (deck *Deck) Sort(repository CardCatalog) error {
	if repository == nil {
		return fmt.Errorf("card repository cannot be nil")
	}
	deck.EnsureOrder()

	mainEntries, mainOrder, err := sortedDeckZone(
		deck.MainDeck,
		deck.MainOrder,
		repository,
	)
	if err != nil {
		return fmt.Errorf("sort main deck: %w", err)
	}
	sideEntries, sideOrder, err := sortedDeckZone(
		deck.SideDeck,
		deck.SideOrder,
		repository,
	)
	if err != nil {
		return fmt.Errorf("sort side deck: %w", err)
	}

	deck.MainDeck = mainEntries
	deck.MainOrder = mainOrder
	deck.SideDeck = sideEntries
	deck.SideOrder = sideOrder
	return nil
}

// sortedDeckZone returns consistently sorted aggregate entries and per-copy order for one zone.
func sortedDeckZone(
	entries []DeckEntry,
	order []string,
	repository CardCatalog,
) ([]DeckEntry, []string, error) {
	entryCards := make([]gamecards.Card, len(entries))
	quantities := make(map[string]int, len(entries))
	for index, entry := range entries {
		card, found := repository.FindByID(entry.CardID)
		if !found {
			return nil, nil, fmt.Errorf("unknown card ID %q", entry.CardID)
		}
		entryCards[index] = card
		quantities[entry.CardID] = entry.Quantity
	}
	gamecards.SortForSearch(entryCards)

	sortedEntries := make([]DeckEntry, len(entryCards))
	for index, card := range entryCards {
		sortedEntries[index] = DeckEntry{
			CardID:   card.ID,
			Quantity: quantities[card.ID],
		}
	}

	orderedCards := make([]gamecards.Card, len(order))
	for index, cardID := range order {
		card, found := repository.FindByID(cardID)
		if !found {
			return nil, nil, fmt.Errorf("unknown card ID %q", cardID)
		}
		orderedCards[index] = card
	}
	gamecards.SortForSearch(orderedCards)

	sortedOrder := make([]string, len(orderedCards))
	for index, card := range orderedCards {
		sortedOrder[index] = card.ID
	}
	return sortedEntries, sortedOrder, nil
}

// orderForZone resolves a zone to its mutable per-copy order slice.
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

// insertCardID inserts one identifier at a clamped position in a per-copy order slice.
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

// MoveOrderedCard reorders one physical copy or transfers it between zones within size limits.
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

		if toIndex < 0 {
			toIndex = 0
		}
		if toIndex > len(updated) {
			toIndex = len(updated)
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

/* MoveOrderedCards creates a list of indices to move that dynamically keeps track of the current cards selected with ctrl/cmd+click and moves them to target index */
func (deck *Deck) MoveOrderedCards(
	fromZone Zone,
	fromIndices []int,
	toZone Zone,
	toIndex int,
) (bool, error) {
	deck.EnsureOrder()

	source, err := deck.orderForZone(fromZone)
	if err != nil {
		return false, err
	}

	indicesCopy := make([]int, len(fromIndices))
	copy(indicesCopy, fromIndices)
	slices.Sort(indicesCopy)
	for index := range len(indicesCopy) {
		if index != 0 && indicesCopy[index] == indicesCopy[index-1] {
			return false, fmt.Errorf("invalid index array")
		}
	}

	if len(indicesCopy) == 0 {
		return false, nil
	}
	for index := range len(indicesCopy) {
		if indicesCopy[index] < 0 || indicesCopy[index] >= len(*source) {
			return false, fmt.Errorf("source index %d is outside deck", indicesCopy[index])
		}
	}

	selectedIDs := []string{}
	remainingIDs := []string{}
	selectionCursor := 0

	for sourceIndex, cardID := range *source {
		if selectionCursor < len(indicesCopy) && sourceIndex == indicesCopy[selectionCursor] {
			selectedIDs = append(selectedIDs, cardID)
			selectionCursor++
		} else {
			remainingIDs = append(remainingIDs, cardID)
		}
	}

	if fromZone == toZone {
		if toIndex < 0 {
			toIndex = 0
		}
		if toIndex > len(remainingIDs) {
			toIndex = len(remainingIDs)
		}

		resultSlice := []string{}
		resultSlice = append(resultSlice, remainingIDs[:toIndex]...)
		resultSlice = append(resultSlice, selectedIDs...)
		resultSlice = append(resultSlice, remainingIDs[toIndex:]...)

		*source = resultSlice
		return true, nil
	}
	return false, fmt.Errorf("cross-zone batch movement is not implemented")
}

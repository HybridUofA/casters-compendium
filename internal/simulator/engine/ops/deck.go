package ops

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// PeekDeckTop returns the top count deck cards without mutating state.
func PeekDeckTop(state *model.MatchState, playerID model.PlayerID, count int) ([]model.MatchCardID, error) {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return nil, err
	}
	if count < 0 {
		return nil, fmt.Errorf("count cannot be negative: %d", count)
	}
	deck := state.Players[index].Deck
	if count > len(deck) {
		return nil, fmt.Errorf("count cannot be greater than amount of cards in deck (%d): %d", len(deck), count)
	}
	return slices.Clone(deck[:count]), nil
}

// PutDeckTopOnBottom moves the top card of playerID's deck to the bottom.
func PutDeckTopOnBottom(state *model.MatchState, playerID model.PlayerID) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	deck := state.Players[index].Deck
	if len(deck) == 0 {
		return fmt.Errorf("deck is empty")
	}
	top := deck[0]
	rest := slices.Clone(deck[1:])
	state.Players[index].Deck = append(rest, top)
	return nil
}

// PutDeckCardOnBottom moves cardID within playerID's deck to the bottom.
func PutDeckCardOnBottom(state *model.MatchState, playerID model.PlayerID, cardID model.MatchCardID) error {
	return moveDeckCard(state, playerID, cardID, model.DeckPlacementBottom)
}

// PutDeckCardOnTop moves cardID within playerID's deck to the top.
func PutDeckCardOnTop(state *model.MatchState, playerID model.PlayerID, cardID model.MatchCardID) error {
	return moveDeckCard(state, playerID, cardID, model.DeckPlacementTop)
}

func moveDeckCard(
	state *model.MatchState,
	playerID model.PlayerID,
	cardID model.MatchCardID,
	placement model.DeckPlacement,
) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	deck := state.Players[index].Deck
	cardIndex := slices.Index(deck, cardID)
	if cardIndex == -1 {
		return fmt.Errorf("card %q is not in %q's deck", cardID, playerID)
	}
	deck = slices.Delete(deck, cardIndex, cardIndex+1)
	switch placement {
	case model.DeckPlacementTop:
		state.Players[index].Deck = append([]model.MatchCardID{cardID}, deck...)
	case model.DeckPlacementBottom:
		state.Players[index].Deck = append(deck, cardID)
	default:
		return fmt.Errorf("unsupported deck placement %q", placement)
	}
	return nil
}

// MoveFromDeckToHand moves a specific card from playerID's deck into their hand.
// Use after a search/choice resolves to a concrete MatchCardID.
func MoveFromDeckToHand(state *model.MatchState, playerID model.PlayerID, cardID model.MatchCardID) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	deck := state.Players[index].Deck
	cardIndex := slices.Index(deck, cardID)
	if cardIndex == -1 {
		return fmt.Errorf("card %q is not in %q's deck", cardID, playerID)
	}
	state.Players[index].Deck = slices.Delete(deck, cardIndex, cardIndex+1)
	state.Players[index].Hand = append(state.Players[index].Hand, cardID)
	return nil
}

// RandomSource is the leaf shuffle entropy interface (engine.RandomSource satisfies it).
type RandomSource interface {
	RandInt(upperBound int) int
}

// ShuffleDeck randomizes playerID's deck order (CR 10.15). Does not bump Revision.
func ShuffleDeck(state *model.MatchState, playerID model.PlayerID, random RandomSource) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	if random == nil {
		return fmt.Errorf("random source cannot be nil")
	}
	deck := state.Players[index].Deck
	for i := len(deck) - 1; i > 0; i-- {
		j := random.RandInt(i + 1)
		if j < 0 || j > i {
			return fmt.Errorf("%d is an invalid shuffle index; must be in [0, %d]", j, i)
		}
		deck[i], deck[j] = deck[j], deck[i]
	}
	state.Players[index].Deck = deck
	return nil
}

// ReorderDeckTops replaces the top len(ordered) cards with ordered.
// ordered must be a permutation of the current top N cards (Emi / dig reorder).
func ReorderDeckTops(state *model.MatchState, playerID model.PlayerID, ordered []model.MatchCardID) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	n := len(ordered)
	if n == 0 {
		return nil
	}
	deck := state.Players[index].Deck
	if n > len(deck) {
		return fmt.Errorf("cannot reorder %d cards; deck only has %d", n, len(deck))
	}
	top := deck[:n]
	counts := map[model.MatchCardID]int{}
	for _, id := range top {
		counts[id]++
	}
	for _, id := range ordered {
		counts[id]--
		if counts[id] < 0 {
			return fmt.Errorf("card %q is not among the top %d deck cards", id, n)
		}
	}
	for id, left := range counts {
		if left != 0 {
			return fmt.Errorf("ordered list is not a permutation of the top %d (missing %q)", n, id)
		}
	}
	newDeck := append(slices.Clone(ordered), slices.Clone(deck[n:])...)
	state.Players[index].Deck = newDeck
	return nil
}

// ReturnToDeck moves cardID from its current zone onto its owner's deck.
func ReturnToDeck(state *model.MatchState, cardID model.MatchCardID, placement model.DeckPlacement) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if placement != model.DeckPlacementTop && placement != model.DeckPlacementBottom {
		return fmt.Errorf("unsupported deck placement %q", placement)
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone == model.ZoneDeck {
		return fmt.Errorf("card %q is already in a deck", cardID)
	}
	if err := removeCardAt(location, state); err != nil {
		return err
	}
	ownerIndex, err := playerIndex(state, instance.Owner)
	if err != nil {
		return fmt.Errorf("owner of %q: %w", cardID, err)
	}
	clearFieldStatus(&instance)
	instance.Face = model.CardFaceDown
	deck := state.Players[ownerIndex].Deck
	switch placement {
	case model.DeckPlacementTop:
		state.Players[ownerIndex].Deck = append([]model.MatchCardID{cardID}, deck...)
	case model.DeckPlacementBottom:
		state.Players[ownerIndex].Deck = append(deck, cardID)
	}
	state.CardInstances[cardID] = instance
	return nil
}

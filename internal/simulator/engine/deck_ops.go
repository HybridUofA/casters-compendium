package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func findPlayerIndex(state *model.MatchState, playerID model.PlayerID) (int, error) {
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

// PeekDeckTops returns the top count cards of deckOwner's deck without mutating.
// Callers must only expose the result to the acting player.
func PeekDeckTops(
	state *model.MatchState,
	deckOwnerID model.PlayerID,
	count int,
) ([]model.MatchCardID, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return nil, fmt.Errorf("match is not in progress")
	}
	if count < 1 {
		return nil, fmt.Errorf("peek count must be at least 1")
	}
	ownerIndex, err := findPlayerIndex(state, deckOwnerID)
	if err != nil {
		return nil, err
	}
	deck := state.Players[ownerIndex].Deck
	if len(deck) == 0 {
		return nil, fmt.Errorf("deck is empty")
	}
	if count > len(deck) {
		count = len(deck)
	}
	return slices.Clone(deck[:count]), nil
}

// MoveDeckTopToBottom moves the top card of deckOwner's deck to the bottom.
// Used for Hilde Willow-style "you may put that card on the bottom".
func MoveDeckTopToBottom(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	deckOwnerID model.PlayerID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if _, err := findPlayerIndex(state, actingPlayerID); err != nil {
		return err
	}
	ownerIndex, err := findPlayerIndex(state, deckOwnerID)
	if err != nil {
		return err
	}
	deck := state.Players[ownerIndex].Deck
	if len(deck) == 0 {
		return fmt.Errorf("deck is empty")
	}
	if len(deck) == 1 {
		state.Revision++
		return nil
	}
	top := deck[0]
	state.Players[ownerIndex].Deck = append(slices.Clone(deck[1:]), top)
	state.Revision++
	return nil
}

// ResolveDeckDig keeps one peeked top card into the acting player's hand and
// puts the remaining peeked cards on the bottom in bottomOrder (first entry is
// nearer the top of that bottom packet). tops must still be the current top of
// deckOwner's deck. This is the Sage Advice dig shape.
func ResolveDeckDig(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	deckOwnerID model.PlayerID,
	keep model.MatchCardID,
	bottomOrder []model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if strings.TrimSpace(string(keep)) == "" {
		return fmt.Errorf("kept card cannot be empty")
	}
	actingIndex, err := findPlayerIndex(state, actingPlayerID)
	if err != nil {
		return err
	}
	ownerIndex, err := findPlayerIndex(state, deckOwnerID)
	if err != nil {
		return err
	}
	if actingPlayerID != deckOwnerID {
		return fmt.Errorf("dig keep must be taken from your own deck")
	}
	pendingSage := state.PendingDraw.Step == model.PendingDrawDig &&
		state.PendingDraw.PlayerID == actingPlayerID
	if state.PendingDraw.Step == model.PendingDrawDig && !pendingSage {
		return fmt.Errorf("pending Sage Advice dig belongs to another player")
	}
	wanted := append([]model.MatchCardID{keep}, bottomOrder...)
	if len(wanted) < 1 {
		return fmt.Errorf("dig requires at least one card")
	}
	deck := state.Players[ownerIndex].Deck
	if len(deck) < len(wanted) {
		return fmt.Errorf("deck no longer has %d cards on top", len(wanted))
	}
	tops := deck[:len(wanted)]
	if !sameCardSet(tops, wanted) {
		return fmt.Errorf("dig cards are not the current top of the deck")
	}
	if !slices.Contains(tops, keep) {
		return fmt.Errorf("kept card is not among the peeked tops")
	}
	for _, cardID := range bottomOrder {
		if cardID == keep {
			return fmt.Errorf("kept card cannot also be put on the bottom")
		}
	}
	state.Players[ownerIndex].Deck = append(slices.Clone(deck[len(wanted):]), slices.Clone(bottomOrder)...)
	state.Players[actingIndex].Hand = append(state.Players[actingIndex].Hand, keep)
	if pendingSage {
		return finishPendingDig(state, catalog)
	}
	state.Revision++
	return nil
}

func sameCardSet(left, right []model.MatchCardID) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[model.MatchCardID]int, len(left))
	for _, id := range left {
		counts[id]++
	}
	for _, id := range right {
		counts[id]--
		if counts[id] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

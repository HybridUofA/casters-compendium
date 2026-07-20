package decks

import (
	"reflect"
	"testing"
)

// TestMoveOrderedCardsReordersWithinZone verifies selected physical copies move as one ordered batch.
func TestMoveOrderedCardsReordersWithinZone(t *testing.T) {
	tests := []struct {
		name      string
		zone      Zone
		indices   []int
		toIndex   int
		wantOrder []string
	}{
		{
			name:      "noncontiguous cards to beginning",
			zone:      MainZone,
			indices:   []int{1, 3},
			toIndex:   0,
			wantOrder: []string{"B", "D", "A", "C", "E"},
		},
		{
			name:      "noncontiguous cards to middle",
			zone:      MainZone,
			indices:   []int{1, 3},
			toIndex:   2,
			wantOrder: []string{"A", "C", "B", "D", "E"},
		},
		{
			name:      "noncontiguous cards to end",
			zone:      MainZone,
			indices:   []int{0, 2},
			toIndex:   3,
			wantOrder: []string{"B", "D", "E", "A", "C"},
		},
		{
			name:      "unsorted input preserves source order",
			zone:      MainZone,
			indices:   []int{3, 1},
			toIndex:   1,
			wantOrder: []string{"A", "B", "D", "C", "E"},
		},
		{
			name:      "negative destination clamps to beginning",
			zone:      MainZone,
			indices:   []int{1, 3},
			toIndex:   -10,
			wantOrder: []string{"B", "D", "A", "C", "E"},
		},
		{
			name:      "oversized destination clamps to end",
			zone:      MainZone,
			indices:   []int{1, 3},
			toIndex:   100,
			wantOrder: []string{"A", "C", "E", "B", "D"},
		},
		{
			name:      "side deck reordering",
			zone:      SideZone,
			indices:   []int{0, 2},
			toIndex:   1,
			wantOrder: []string{"Y", "X", "Z"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deck := orderedMoveTestDeck()
			indicesBefore := append([]int(nil), test.indices...)
			mainEntriesBefore := append([]DeckEntry(nil), deck.MainDeck...)
			sideEntriesBefore := append([]DeckEntry(nil), deck.SideDeck...)

			moved, err := deck.MoveOrderedCards(
				test.zone,
				test.indices,
				test.zone,
				test.toIndex,
			)
			if err != nil {
				t.Fatal(err)
			}
			if !moved {
				t.Fatal("MoveOrderedCards reported that no cards moved")
			}

			gotOrder := deck.MainOrder
			if test.zone == SideZone {
				gotOrder = deck.SideOrder
			}
			if !reflect.DeepEqual(gotOrder, test.wantOrder) {
				t.Fatalf("zone order = %#v, want %#v", gotOrder, test.wantOrder)
			}
			if !reflect.DeepEqual(test.indices, indicesBefore) {
				t.Fatalf("input indices changed from %#v to %#v", indicesBefore, test.indices)
			}
			if !reflect.DeepEqual(deck.MainDeck, mainEntriesBefore) {
				t.Fatalf("main-deck entries changed during reorder: %#v", deck.MainDeck)
			}
			if !reflect.DeepEqual(deck.SideDeck, sideEntriesBefore) {
				t.Fatalf("side-deck entries changed during reorder: %#v", deck.SideDeck)
			}
		})
	}
}

// TestMoveOrderedCardsRejectsInvalidSelections verifies failed requests do not mutate the deck.
func TestMoveOrderedCardsRejectsInvalidSelections(t *testing.T) {
	tests := []struct {
		name    string
		zone    Zone
		indices []int
	}{
		{name: "duplicate index", zone: MainZone, indices: []int{1, 1}},
		{name: "negative index", zone: MainZone, indices: []int{-1}},
		{name: "out-of-range index", zone: MainZone, indices: []int{5}},
		{name: "unknown source zone", zone: Zone("unknown"), indices: []int{0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deck := orderedMoveTestDeck()
			before := cloneOrderedMoveTestDeck(deck)

			moved, err := deck.MoveOrderedCards(
				test.zone,
				test.indices,
				test.zone,
				0,
			)
			if err == nil {
				t.Fatal("MoveOrderedCards returned no error for invalid input")
			}
			if moved {
				t.Fatal("MoveOrderedCards reported movement for invalid input")
			}
			if !reflect.DeepEqual(deck, before) {
				t.Fatalf("deck changed after rejected move: %#v", deck)
			}
		})
	}
}

// TestMoveOrderedCardsEmptySelectionIsNoOp verifies empty selections are harmless.
func TestMoveOrderedCardsEmptySelectionIsNoOp(t *testing.T) {
	deck := orderedMoveTestDeck()
	before := cloneOrderedMoveTestDeck(deck)

	moved, err := deck.MoveOrderedCards(MainZone, nil, MainZone, 2)
	if err != nil {
		t.Fatal(err)
	}
	if moved {
		t.Fatal("MoveOrderedCards reported movement for an empty selection")
	}
	if !reflect.DeepEqual(deck, before) {
		t.Fatalf("deck changed after empty selection: %#v", deck)
	}
}

// TestMoveOrderedCardsRejectsCrossZoneUntilImplemented verifies the partial implementation fails safely.
func TestMoveOrderedCardsRejectsCrossZoneUntilImplemented(t *testing.T) {
	deck := orderedMoveTestDeck()
	before := cloneOrderedMoveTestDeck(deck)

	moved, err := deck.MoveOrderedCards(MainZone, []int{1, 3}, SideZone, 1)
	if err == nil {
		t.Fatal("MoveOrderedCards returned no error for an unimplemented cross-zone move")
	}
	if moved {
		t.Fatal("MoveOrderedCards reported movement for an unimplemented cross-zone move")
	}
	if !reflect.DeepEqual(deck, before) {
		t.Fatalf("deck changed after rejected cross-zone move: %#v", deck)
	}
}

func orderedMoveTestDeck() *Deck {
	return &Deck{
		SchemaVersion: 1,
		Name:          "Ordered Move Test",
		MainDeck: []DeckEntry{
			{CardID: "A", Quantity: 1},
			{CardID: "B", Quantity: 1},
			{CardID: "C", Quantity: 1},
			{CardID: "D", Quantity: 1},
			{CardID: "E", Quantity: 1},
		},
		SideDeck: []DeckEntry{
			{CardID: "X", Quantity: 1},
			{CardID: "Y", Quantity: 1},
			{CardID: "Z", Quantity: 1},
		},
		MainOrder: []string{"A", "B", "C", "D", "E"},
		SideOrder: []string{"X", "Y", "Z"},
	}
}

func cloneOrderedMoveTestDeck(deck *Deck) *Deck {
	cloned := *deck
	cloned.MainDeck = append([]DeckEntry(nil), deck.MainDeck...)
	cloned.SideDeck = append([]DeckEntry(nil), deck.SideDeck...)
	cloned.MainOrder = append([]string(nil), deck.MainOrder...)
	cloned.SideOrder = append([]string(nil), deck.SideOrder...)
	return &cloned
}

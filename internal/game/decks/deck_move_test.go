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

// TestMoveOrderedCardsMovesAcrossZones verifies both transfer directions and insertion clamping.
func TestMoveOrderedCardsMovesAcrossZones(t *testing.T) {
	tests := []struct {
		name          string
		fromZone      Zone
		indices       []int
		toZone        Zone
		toIndex       int
		wantMainOrder []string
		wantSideOrder []string
	}{
		{
			name:          "main to side preserves source order",
			fromZone:      MainZone,
			indices:       []int{3, 1},
			toZone:        SideZone,
			toIndex:       1,
			wantMainOrder: []string{"A", "C", "E"},
			wantSideOrder: []string{"X", "B", "D", "Y", "Z"},
		},
		{
			name:          "side to main",
			fromZone:      SideZone,
			indices:       []int{0, 2},
			toZone:        MainZone,
			toIndex:       2,
			wantMainOrder: []string{"A", "B", "X", "Z", "C", "D", "E"},
			wantSideOrder: []string{"Y"},
		},
		{
			name:          "negative destination clamps to beginning",
			fromZone:      MainZone,
			indices:       []int{1},
			toZone:        SideZone,
			toIndex:       -20,
			wantMainOrder: []string{"A", "C", "D", "E"},
			wantSideOrder: []string{"B", "X", "Y", "Z"},
		},
		{
			name:          "oversized destination clamps to end",
			fromZone:      SideZone,
			indices:       []int{1},
			toZone:        MainZone,
			toIndex:       200,
			wantMainOrder: []string{"A", "B", "C", "D", "E", "Y"},
			wantSideOrder: []string{"X", "Z"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deck := orderedMoveTestDeck()
			indicesBefore := append([]int(nil), test.indices...)
			totalBefore := deck.TotalCards()

			moved, err := deck.MoveOrderedCards(
				test.fromZone,
				test.indices,
				test.toZone,
				test.toIndex,
			)
			if err != nil {
				t.Fatal(err)
			}
			if !moved {
				t.Fatal("MoveOrderedCards reported that no cards moved")
			}
			if !reflect.DeepEqual(deck.MainOrder, test.wantMainOrder) {
				t.Fatalf("main order = %#v, want %#v", deck.MainOrder, test.wantMainOrder)
			}
			if !reflect.DeepEqual(deck.SideOrder, test.wantSideOrder) {
				t.Fatalf("side order = %#v, want %#v", deck.SideOrder, test.wantSideOrder)
			}
			if !reflect.DeepEqual(test.indices, indicesBefore) {
				t.Fatalf("input indices changed from %#v to %#v", indicesBefore, test.indices)
			}
			if deck.TotalCards() != totalBefore {
				t.Fatalf("total cards = %d, want conserved total %d", deck.TotalCards(), totalBefore)
			}
			assertOrderedMoveDeckConsistent(t, deck)
		})
	}
}

// TestMoveOrderedCardsTransfersRepeatedIDs verifies physical copies update aggregate quantities.
func TestMoveOrderedCardsTransfersRepeatedIDs(t *testing.T) {
	deck := &Deck{
		MainDeck: []DeckEntry{
			{CardID: "A", Quantity: 3},
			{CardID: "B", Quantity: 1},
			{CardID: "C", Quantity: 1},
		},
		SideDeck:  []DeckEntry{{CardID: "A", Quantity: 1}, {CardID: "X", Quantity: 1}},
		MainOrder: []string{"A", "B", "A", "C", "A"},
		SideOrder: []string{"A", "X"},
	}

	moved, err := deck.MoveOrderedCards(MainZone, []int{4, 0}, SideZone, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !moved {
		t.Fatal("MoveOrderedCards reported that no cards moved")
	}
	if !reflect.DeepEqual(deck.MainOrder, []string{"B", "A", "C"}) {
		t.Fatalf("main order = %#v", deck.MainOrder)
	}
	if !reflect.DeepEqual(deck.SideOrder, []string{"A", "A", "A", "X"}) {
		t.Fatalf("side order = %#v", deck.SideOrder)
	}
	if quantity, _ := deck.QuantityInZone(MainZone, "A"); quantity != 1 {
		t.Fatalf("main A quantity = %d, want 1", quantity)
	}
	if quantity, _ := deck.QuantityInZone(SideZone, "A"); quantity != 3 {
		t.Fatalf("side A quantity = %d, want 3", quantity)
	}
	assertOrderedMoveDeckConsistent(t, deck)
}

// TestMoveOrderedCardsHonorsCrossZoneCapacity verifies exact capacity succeeds and overflow is atomic.
func TestMoveOrderedCardsHonorsCrossZoneCapacity(t *testing.T) {
	t.Run("exact side capacity succeeds", func(t *testing.T) {
		deck := orderedMoveTestDeck()
		deck.SideDeck = []DeckEntry{{CardID: "X", Quantity: 10}}
		deck.SideOrder = repeatedCardIDs("X", 10)

		moved, err := deck.MoveOrderedCards(MainZone, []int{1, 3}, SideZone, 10)
		if err != nil {
			t.Fatal(err)
		}
		if !moved {
			t.Fatal("exact-capacity move was rejected")
		}
		if deck.SideTotal() != MaxSideDeckCards {
			t.Fatalf("side total = %d, want %d", deck.SideTotal(), MaxSideDeckCards)
		}
		assertOrderedMoveDeckConsistent(t, deck)
	})

	t.Run("side overflow changes nothing", func(t *testing.T) {
		deck := orderedMoveTestDeck()
		deck.SideDeck = []DeckEntry{{CardID: "X", Quantity: 11}}
		deck.SideOrder = repeatedCardIDs("X", 11)
		before := cloneOrderedMoveTestDeck(deck)

		moved, err := deck.MoveOrderedCards(MainZone, []int{1, 3}, SideZone, 0)
		if err != nil {
			t.Fatal(err)
		}
		if moved {
			t.Fatal("overflowing move reported success")
		}
		if !reflect.DeepEqual(deck, before) {
			t.Fatalf("deck changed after capacity rejection: %#v", deck)
		}
	})

	t.Run("main overflow changes nothing", func(t *testing.T) {
		deck := orderedMoveTestDeck()
		deck.MainDeck = []DeckEntry{{CardID: "A", Quantity: MaxMainDeckCards}}
		deck.MainOrder = repeatedCardIDs("A", MaxMainDeckCards)
		before := cloneOrderedMoveTestDeck(deck)

		moved, err := deck.MoveOrderedCards(SideZone, []int{0}, MainZone, 0)
		if err != nil {
			t.Fatal(err)
		}
		if moved {
			t.Fatal("overflowing move reported success")
		}
		if !reflect.DeepEqual(deck, before) {
			t.Fatalf("deck changed after capacity rejection: %#v", deck)
		}
	})
}

// TestMoveOrderedCardsCrossZoneFailuresAreAtomic verifies invalid requests never partially transfer cards.
func TestMoveOrderedCardsCrossZoneFailuresAreAtomic(t *testing.T) {
	t.Run("unknown destination zone", func(t *testing.T) {
		deck := orderedMoveTestDeck()
		before := cloneOrderedMoveTestDeck(deck)

		moved, err := deck.MoveOrderedCards(MainZone, []int{0}, Zone("unknown"), 0)
		if err == nil {
			t.Fatal("MoveOrderedCards returned no error for an unknown destination zone")
		}
		if moved {
			t.Fatal("invalid destination move reported success")
		}
		if !reflect.DeepEqual(deck, before) {
			t.Fatalf("deck changed after invalid destination: %#v", deck)
		}
	})

	t.Run("aggregate update error", func(t *testing.T) {
		deck := &Deck{
			MainDeck:  []DeckEntry{{CardID: "", Quantity: 1}},
			SideDeck:  []DeckEntry{{CardID: "X", Quantity: 1}},
			MainOrder: []string{""},
			SideOrder: []string{"X"},
		}
		before := cloneOrderedMoveTestDeck(deck)

		moved, err := deck.MoveOrderedCards(MainZone, []int{0}, SideZone, 1)
		if err == nil {
			t.Fatal("MoveOrderedCards returned no error for an invalid card ID")
		}
		if moved {
			t.Fatal("failed aggregate update reported success")
		}
		if !reflect.DeepEqual(deck, before) {
			t.Fatalf("deck changed after aggregate update failed: %#v", deck)
		}
	})
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

func repeatedCardIDs(cardID string, quantity int) []string {
	result := make([]string, quantity)
	for index := range result {
		result[index] = cardID
	}
	return result
}

func assertOrderedMoveDeckConsistent(t *testing.T, deck *Deck) {
	t.Helper()

	assertZone := func(zone Zone, order []string, entries []DeckEntry) {
		t.Helper()
		orderCounts := make(map[string]int)
		for _, cardID := range order {
			orderCounts[cardID]++
		}
		entryCounts := make(map[string]int)
		for _, entry := range entries {
			entryCounts[entry.CardID] += entry.Quantity
		}
		if !reflect.DeepEqual(orderCounts, entryCounts) {
			t.Fatalf("%s order counts %#v do not match entries %#v", zone, orderCounts, entryCounts)
		}
	}

	assertZone(MainZone, deck.MainOrder, deck.MainDeck)
	assertZone(SideZone, deck.SideOrder, deck.SideDeck)
}

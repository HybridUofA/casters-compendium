package deckbuilder

import (
	"slices"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	deckui "github.com/HybridUofA/casters-compendium/internal/deckbuilder/ui"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// TestApplyCardDropRemovesDeckCard verifies that the search-panel drop target removes one physical copy.
func TestApplyCardDropRemovesDeckCard(t *testing.T) {
	repository, err := catalog.NewRepository([]catalog.Card{
		{ID: "test-card", Name: "Test Card"},
	})
	if err != nil {
		t.Fatal(err)
	}

	deck, err := decks.NewDeck("Drag Removal Test")
	if err != nil {
		t.Fatal(err)
	}

	card, found := repository.FindByID("test-card")
	if !found {
		t.Fatal("test card was not added to the repository")
	}

	added, err := deck.AddCardChecked(
		decks.MainZone,
		card,
		2,
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("test card was not added to the deck")
	}

	err = applyCardDrop(
		deck,
		repository,
		deckui.CardDragSource{
			Kind:  deckui.DragFromDeck,
			Card:  card,
			Zone:  decks.MainZone,
			Index: 0,
		},
		&deckui.CardDropTarget{Remove: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := deck.MainTotal(); got != 1 {
		t.Fatalf("main-deck total = %d, want 1", got)
	}
	if got := len(deck.MainOrder); got != 1 {
		t.Fatalf("main-deck order length = %d, want 1", got)
	}
}

// TestApplyCardDropIgnoresSearchCardRemoval verifies that search results cannot remove deck cards.
func TestApplyCardDropIgnoresSearchCardRemoval(t *testing.T) {
	repository, err := catalog.NewRepository([]catalog.Card{
		{ID: "test-card", Name: "Test Card"},
	})
	if err != nil {
		t.Fatal(err)
	}

	deck, err := decks.NewDeck("Search Source Test")
	if err != nil {
		t.Fatal(err)
	}

	card, found := repository.FindByID("test-card")
	if !found {
		t.Fatal("test card was not added to the repository")
	}

	if err := applyCardDrop(
		deck,
		repository,
		deckui.CardDragSource{
			Kind: deckui.DragFromSearch,
			Card: card,
		},
		&deckui.CardDropTarget{Remove: true},
	); err != nil {
		t.Fatal(err)
	}

	if got := deck.MainTotal(); got != 0 {
		t.Fatalf("main-deck total = %d, want 0", got)
	}
}

// TestApplyCardDropMovesSelectedDeckCards verifies the application forwards complete batches.
func TestApplyCardDropMovesSelectedDeckCards(t *testing.T) {
	tests := []struct {
		name          string
		source        deckui.CardDragSource
		target        *deckui.CardDropTarget
		wantMainOrder []string
		wantSideOrder []string
	}{
		{
			name: "same-zone batch",
			source: deckui.CardDragSource{
				Kind:    deckui.DragFromDeck,
				Zone:    decks.MainZone,
				Index:   3,
				Indices: []int{3, 1},
			},
			target:        &deckui.CardDropTarget{Zone: decks.MainZone, Index: 1},
			wantMainOrder: []string{"A", "B", "D", "C", "E"},
			wantSideOrder: []string{"X", "Y"},
		},
		{
			name: "cross-zone batch",
			source: deckui.CardDragSource{
				Kind:    deckui.DragFromDeck,
				Zone:    decks.MainZone,
				Index:   1,
				Indices: []int{1, 3},
			},
			target:        &deckui.CardDropTarget{Zone: decks.SideZone, Index: 1},
			wantMainOrder: []string{"A", "C", "E"},
			wantSideOrder: []string{"X", "B", "D", "Y"},
		},
		{
			name: "empty batch falls back to single index",
			source: deckui.CardDragSource{
				Kind:    deckui.DragFromDeck,
				Zone:    decks.MainZone,
				Index:   2,
				Indices: []int{},
			},
			target:        &deckui.CardDropTarget{Zone: decks.MainZone, Index: 0},
			wantMainOrder: []string{"C", "A", "B", "D", "E"},
			wantSideOrder: []string{"X", "Y"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deck := batchDropTestDeck()
			indicesBefore := slices.Clone(test.source.Indices)

			if err := applyCardDrop(deck, nil, test.source, test.target); err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(deck.MainOrder, test.wantMainOrder) {
				t.Fatalf("main order = %#v, want %#v", deck.MainOrder, test.wantMainOrder)
			}
			if !slices.Equal(deck.SideOrder, test.wantSideOrder) {
				t.Fatalf("side order = %#v, want %#v", deck.SideOrder, test.wantSideOrder)
			}
			if !slices.Equal(test.source.Indices, indicesBefore) {
				t.Fatalf("source indices changed from %#v to %#v", indicesBefore, test.source.Indices)
			}
		})
	}
}

func batchDropTestDeck() *decks.Deck {
	return &decks.Deck{
		MainDeck: []decks.DeckEntry{
			{CardID: "A", Quantity: 1},
			{CardID: "B", Quantity: 1},
			{CardID: "C", Quantity: 1},
			{CardID: "D", Quantity: 1},
			{CardID: "E", Quantity: 1},
		},
		SideDeck: []decks.DeckEntry{
			{CardID: "X", Quantity: 1},
			{CardID: "Y", Quantity: 1},
		},
		MainOrder: []string{"A", "B", "C", "D", "E"},
		SideOrder: []string{"X", "Y"},
	}
}
